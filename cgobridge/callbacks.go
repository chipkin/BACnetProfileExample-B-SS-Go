// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.
//
// callbacks.go - the stack's callbacks, and the BACnet/IP transport behind them
// =============================================================================
// WORKAROUND - ADAPTER DEFECT, cas-bacnet-stack#3968 (see TODO.md). The
// callback trampolines below and in trampolines.c are what the stack's Go
// adapter would supply if it compiled at the pin. Each callback has three
// layers, because cgo cannot hand a Go function to C directly:
//
//   the stack -> fpCallbackXxx (C, trampolines.c, the header's exact
//   signature) -> goCallbackXxx (Go, //export'ed below) -> the Go function
//   main.go registered.
//
// The registrations are made in C (cgobridge_register* in trampolines.c), where
// the compiler checks each trampoline against the adapter's generated
// callback type - a signature change in the stack's header fails the build.
//
// This file's cgo preamble holds DECLARATIONS ONLY: cgo copies the preamble of
// a file that has //export functions into _cgo_export.c as well, so a
// function defined here would be defined twice. The C definitions are in
// trampolines.c, compiled once.
//
// The transport half (receive, send, time, the UDP socket and the start-up
// I-Am) is the Go edition of the C++ common/CASExampleHelper's
// RegisterCommonCallbacks() / SetupUDP() / SendIAm(). It lives here, not in
// common/, because an //export'ed function has to be in the package with the
// `import "C"`.
// =============================================================================

package cgobridge

/*
#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include "CASBACnetStackAdapterTypes.h"

extern void cgobridge_registerCommonCallbacks(void);
extern void cgobridge_registerGetPropertyReal(void);
extern void cgobridge_registerGetPropertyEnumerated(void);
extern void cgobridge_registerGetPropertyUnsignedInteger(void);
extern void cgobridge_registerGetPropertyCharString(void);
extern void cgobridge_registerGetPropertyBool(void);
extern void cgobridge_registerGetPropertyOctetString(void);

*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/chipkin/bacnet-profile-example-b-ss-go/common"
)

// -----------------------------------------------------------------------------
// Transport: the UDP socket, the receive/send/time callbacks, the start-up I-Am
// -----------------------------------------------------------------------------

var (
	udp                 common.SimpleUDP
	boundPort           uint16
	networkPortInstance uint32 = 1
	localAddressing     common.LocalIPv4
)

// SetNetworkPortInstance names the Network Port object that owns the socket.
// The stack identifies a link by its Network Port INSTANCE, so the transport
// callbacks and the start-up I-Am have to name it.
func SetNetworkPortInstance(instance uint32) { networkPortInstance = instance }

// SetupUDP binds the BACnet/IP socket and reads the primary IPv4 interface's
// addressing (for the subnet broadcast).
func SetupUDP(port uint16) bool {
	if err := udp.Setup(port); err != nil {
		fmt.Printf("Error: Failed to bind UDP port %d (Network Port %d).\n", port, networkPortInstance)
		return false
	}
	boundPort = port
	localAddressing = common.GetLocalIPv4()
	fmt.Printf("FYI: Listening for BACnet/IP on UDP port %d (Network Port %d).\n", port, networkPortInstance)
	return true
}

// ShutdownUDP closes the socket.
func ShutdownUDP() { udp.Shutdown() }

// RegisterCommonCallbacks registers the transport and time callbacks.
func RegisterCommonCallbacks() {
	C.cgobridge_registerCommonCallbacks()
}

// SendIAm broadcasts an unsolicited I-Am to the LOCAL subnet broadcast (the
// device's own network, ip | ~mask) on the bound port - not the global
// 255.255.255.255 / network 0xFFFF.
func SendIAm(deviceInstance uint32) bool {
	broadcast := localAddressing.Broadcast
	connectionString := [6]byte{broadcast[0], broadcast[1], broadcast[2], broadcast[3], byte(boundPort >> 8), byte(boundPort & 0xFF)}
	return sendIAm(deviceInstance, connectionString, networkPortInstance, true)
}

// goCallbackReceiveMessageForPort: the stack calls this to RECEIVE a message.
// Pull one datagram off the socket; if one arrived, copy it into the stack's
// buffer and report who sent it as a 6-octet connection string (IP[4] +
// port[2], big-endian). Returning 0 means "nothing to process this tick".
//
//export goCallbackReceiveMessageForPort
func goCallbackReceiveMessageForPort(message *C.uint8_t, maxMessageLength C.uint16_t,
	sourceConnectionString *C.uint8_t, sourceConnectionStringLength *C.uint8_t,
	destinationConnectionString *C.uint8_t, destinationConnectionStringLength *C.uint8_t,
	maxConnectionStringLength C.uint8_t, receivedOnNetworkPort *C.uint32_t) C.uint16_t {
	if maxConnectionStringLength < 6 {
		return 0
	}
	datagram, ok := udp.Recv()
	if !ok {
		return 0
	}
	// A datagram larger than the stack's buffer is truncated to it (the
	// stack then fails to decode it and ignores it), as in the C++ helper.
	length := len(datagram.Message)
	if length > int(maxMessageLength) {
		length = int(maxMessageLength)
	}
	if length > 0 {
		C.memcpy(unsafe.Pointer(message), unsafe.Pointer(&datagram.Message[0]), C.size_t(length))
	}
	fmt.Printf("RX %d bytes from %d.%d.%d.%d:%d (Network Port %d) - %s\n", length,
		datagram.FromIP[0], datagram.FromIP[1], datagram.FromIP[2], datagram.FromIP[3], datagram.FromPort,
		networkPortInstance, common.SummarizeBacnetFrame(datagram.Message[:length]))

	source := unsafe.Slice((*byte)(unsafe.Pointer(sourceConnectionString)), 6)
	copy(source[:4], datagram.FromIP[:])
	source[4] = byte(datagram.FromPort >> 8)
	source[5] = byte(datagram.FromPort & 0xFF)
	*sourceConnectionStringLength = 6
	// Which Network Port object the datagram arrived on: this device's one.
	*receivedOnNetworkPort = C.uint32_t(networkPortInstance)
	return C.uint16_t(length)
}

// goCallbackSendMessageForPort: the stack calls this to SEND a message to a
// connection string (IP[4] + port[2]), unicast or broadcast alike.
//   - No connection string: "broadcast on this data link" - send to the local
//     subnet broadcast on the bound port (common/ 3.2.0).
//   - 1..5 octets: not a BACnet/IP address - refused (return 0).
//   - broadcast = true with THIS socket's own address: the stack means "this
//     subnet" (stack issue #3933) - send to the local subnet broadcast
//     (common/ 3.4.0). Every other address is sent to as given.
//
//export goCallbackSendMessageForPort
func goCallbackSendMessageForPort(message *C.uint8_t, messageLength C.uint16_t,
	connectionString *C.uint8_t, connectionStringLength C.uint8_t,
	leavingByNetworkPort C.uint32_t, broadcast C.bool) C.uint16_t {
	if uint32(leavingByNetworkPort) != networkPortInstance {
		return 0 // not a port this device has a socket for
	}
	var toIP [4]byte
	var toPort uint16
	isBroadcast := bool(broadcast)
	if connectionString == nil || connectionStringLength == 0 {
		toIP = localAddressing.Broadcast
		toPort = boundPort
		isBroadcast = true
	} else if connectionStringLength < 6 {
		return 0
	} else {
		octets := unsafe.Slice((*byte)(unsafe.Pointer(connectionString)), 6)
		copy(toIP[:], octets[:4])
		toPort = uint16(octets[4])<<8 | uint16(octets[5])
		if bool(broadcast) && toPort == boundPort && localAddressing.Found && toIP == localAddressing.Address {
			toIP = localAddressing.Broadcast
		}
	}

	frame := C.GoBytes(unsafe.Pointer(message), C.int(messageLength))
	sent := 0
	if udp.Send(frame, toIP, toPort) {
		sent = len(frame)
	}
	broadcastLabel := ""
	if isBroadcast {
		broadcastLabel = " (broadcast)"
	}
	fmt.Printf("TX %d bytes to %d.%d.%d.%d:%d%s (Network Port %d) - %s\n", sent,
		toIP[0], toIP[1], toIP[2], toIP[3], toPort, broadcastLabel,
		uint32(leavingByNetworkPort), common.SummarizeBacnetFrame(frame))
	return C.uint16_t(sent)
}

// goCallbackGetSystemTime: the current time as CASBACnetTime, int64 seconds
// since the UNIX epoch.
//
//export goCallbackGetSystemTime
func goCallbackGetSystemTime() C.CASBACnetTime {
	return C.CASBACnetTime(time.Now().Unix())
}

// -----------------------------------------------------------------------------
// Get*Property: the stack asks the application for a property's value
// -----------------------------------------------------------------------------
// Each callback is a plain Go function in main (property_dispatch.go) that
// returns (value, true) when it serves the property and (zero, false) when it
// declines. errorCode is the stack's own out-parameter: leave it alone to
// decline with "no opinion", set it to name the BACnet error (main.go section 3
// explains when).

// GetPropertyRealFunc serves REAL properties.
type GetPropertyRealFunc func(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, useArrayIndex bool, propertyArrayIndex uint32, errorCode *uint32) (float32, bool)

// GetPropertyUint32Func serves ENUMERATED or UNSIGNED INTEGER properties.
type GetPropertyUint32Func func(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, useArrayIndex bool, propertyArrayIndex uint32, errorCode *uint32) (uint32, bool)

// GetPropertyBoolFunc serves BOOLEAN properties.
type GetPropertyBoolFunc func(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, useArrayIndex bool, propertyArrayIndex uint32, errorCode *uint32) (bool, bool)

// GetPropertyCharStringFunc serves CHARACTER STRING properties (UTF-8).
type GetPropertyCharStringFunc func(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, useArrayIndex bool, propertyArrayIndex uint32, errorCode *uint32) (string, bool)

// GetPropertyOctetStringFunc serves OCTET STRING properties.
type GetPropertyOctetStringFunc func(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, useArrayIndex bool, propertyArrayIndex uint32, errorCode *uint32) ([]byte, bool)

var (
	getPropertyReal            GetPropertyRealFunc
	getPropertyEnumerated      GetPropertyUint32Func
	getPropertyUnsignedInteger GetPropertyUint32Func
	getPropertyBool            GetPropertyBoolFunc
	getPropertyCharString      GetPropertyCharStringFunc
	getPropertyOctetString     GetPropertyOctetStringFunc
)

// RegisterCallbackGetPropertyReal is BACnetStack_RegisterCallbackGetPropertyReal.
func RegisterCallbackGetPropertyReal(callback GetPropertyRealFunc) {
	getPropertyReal = callback
	C.cgobridge_registerGetPropertyReal()
}

// RegisterCallbackGetPropertyEnumerated is BACnetStack_RegisterCallbackGetPropertyEnumerated.
func RegisterCallbackGetPropertyEnumerated(callback GetPropertyUint32Func) {
	getPropertyEnumerated = callback
	C.cgobridge_registerGetPropertyEnumerated()
}

// RegisterCallbackGetPropertyUnsignedInteger is BACnetStack_RegisterCallbackGetPropertyUnsignedInteger.
func RegisterCallbackGetPropertyUnsignedInteger(callback GetPropertyUint32Func) {
	getPropertyUnsignedInteger = callback
	C.cgobridge_registerGetPropertyUnsignedInteger()
}

// RegisterCallbackGetPropertyCharacterString is BACnetStack_RegisterCallbackGetPropertyCharacterString.
func RegisterCallbackGetPropertyCharacterString(callback GetPropertyCharStringFunc) {
	getPropertyCharString = callback
	C.cgobridge_registerGetPropertyCharString()
}

// RegisterCallbackGetPropertyBool is BACnetStack_RegisterCallbackGetPropertyBool.
func RegisterCallbackGetPropertyBool(callback GetPropertyBoolFunc) {
	getPropertyBool = callback
	C.cgobridge_registerGetPropertyBool()
}

// RegisterCallbackGetPropertyOctetString is BACnetStack_RegisterCallbackGetPropertyOctetString.
func RegisterCallbackGetPropertyOctetString(callback GetPropertyOctetStringFunc) {
	getPropertyOctetString = callback
	C.cgobridge_registerGetPropertyOctetString()
}

//export goCallbackGetPropertyReal
func goCallbackGetPropertyReal(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.float, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	if getPropertyReal == nil {
		return false
	}
	result, ok := getPropertyReal(uint32(deviceInstance), uint16(objectType), uint32(objectInstance), uint32(propertyIdentifier), bool(useArrayIndex), uint32(propertyArrayIndex), (*uint32)(unsafe.Pointer(errorCode)))
	if ok {
		*value = C.float(result)
	}
	return C.bool(ok)
}

//export goCallbackGetPropertyEnumerated
func goCallbackGetPropertyEnumerated(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.uint32_t, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	if getPropertyEnumerated == nil {
		return false
	}
	result, ok := getPropertyEnumerated(uint32(deviceInstance), uint16(objectType), uint32(objectInstance), uint32(propertyIdentifier), bool(useArrayIndex), uint32(propertyArrayIndex), (*uint32)(unsafe.Pointer(errorCode)))
	if ok {
		*value = C.uint32_t(result)
	}
	return C.bool(ok)
}

//export goCallbackGetPropertyUnsignedInteger
func goCallbackGetPropertyUnsignedInteger(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.uint32_t, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	if getPropertyUnsignedInteger == nil {
		return false
	}
	result, ok := getPropertyUnsignedInteger(uint32(deviceInstance), uint16(objectType), uint32(objectInstance), uint32(propertyIdentifier), bool(useArrayIndex), uint32(propertyArrayIndex), (*uint32)(unsafe.Pointer(errorCode)))
	if ok {
		*value = C.uint32_t(result)
	}
	return C.bool(ok)
}

//export goCallbackGetPropertyBool
func goCallbackGetPropertyBool(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.bool, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	if getPropertyBool == nil {
		return false
	}
	result, ok := getPropertyBool(uint32(deviceInstance), uint16(objectType), uint32(objectInstance), uint32(propertyIdentifier), bool(useArrayIndex), uint32(propertyArrayIndex), (*uint32)(unsafe.Pointer(errorCode)))
	if ok {
		*value = C.bool(result)
	}
	return C.bool(ok)
}

// goCallbackGetPropertyCharString copies the string into the stack's buffer.
// A string longer than maxElementCount is truncated SILENTLY to fit (main.go
// section 3, ReturnCharacterString in the C++ reference, says why that never
// trips here).
//
//export goCallbackGetPropertyCharString
func goCallbackGetPropertyCharString(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.char, valueElementCount *C.uint32_t, maxElementCount C.uint32_t, encodingType *C.uint8_t, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	if getPropertyCharString == nil {
		return false
	}
	text, ok := getPropertyCharString(uint32(deviceInstance), uint16(objectType), uint32(objectInstance), uint32(propertyIdentifier), bool(useArrayIndex), uint32(propertyArrayIndex), (*uint32)(unsafe.Pointer(errorCode)))
	if !ok {
		return false
	}
	length := len(text)
	if length > int(maxElementCount) {
		length = int(maxElementCount)
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(value)), length), text[:length])
	*valueElementCount = C.uint32_t(length)
	*encodingType = C.uint8_t(common.CharacterStringEncodingUTF8)
	return true
}

//export goCallbackGetPropertyOctetString
func goCallbackGetPropertyOctetString(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.uint8_t, valueElementCount *C.uint32_t, maxElementCount C.uint32_t, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	if getPropertyOctetString == nil {
		return false
	}
	octets, ok := getPropertyOctetString(uint32(deviceInstance), uint16(objectType), uint32(objectInstance), uint32(propertyIdentifier), bool(useArrayIndex), uint32(propertyArrayIndex), (*uint32)(unsafe.Pointer(errorCode)))
	if !ok || len(octets) > int(maxElementCount) {
		return false
	}
	copy(unsafe.Slice((*byte)(unsafe.Pointer(value)), len(octets)), octets)
	*valueElementCount = C.uint32_t(len(octets))
	return true
}
