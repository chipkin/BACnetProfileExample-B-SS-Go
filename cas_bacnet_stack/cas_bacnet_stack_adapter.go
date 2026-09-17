// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.
//
// cas_bacnet_stack_adapter.go
// =============================================================================
// VENDORED (COPY, not a reference in place) from
// submodules/cas-bacnet-stack/adapters/golang/CASBACnetStackAdapter.go,
// then FIXED to compile against this repo's pinned 6.x header
// (source/CASBACnetStackDLL.h). It shipped broken against the current
// header - see README.md in this directory for the full list of fixes,
// and AGENTS.md/CHANGELOG.md at the repo root for why. NEVER edit the
// submodule's copy - only this vendored one.
//
// Package casbacnetstack is the one package in this repo with
// `import "C"` and cgo //export functions. cgo requires every exported
// callback AND the C preamble that declares its C-side trampoline to live
// in the same package, so all the transport, time and Get*Property
// callbacks - and the thin Go wrappers main.go uses to drive the stack
// (AddDevice, AddObject, Tick, SendIAm, ...) - live here rather than being
// split across common/ the way the Rust edition's cas_example_helper.rs
// is. See this directory's README.md for the fixes made, and
// common/device_state.go's doc comment for why the callbacks below only
// ever read/write package-level state, never a closure.
// =============================================================================

package casbacnetstack

/*
#cgo CFLAGS: -I../submodules/cas-bacnet-stack/source
#cgo windows LDFLAGS: -L../lib -lCASBACnetStack_x64_Release
#cgo linux LDFLAGS: -L../lib -lCASBACnetStack_x64_Release

#include <string.h>
#include "CASBACnetStackDLL.h"

// Go Callback Export Declarations
// ====================================================
// General Callbacks
//
// FIX (#1 in README.md): BACnetStack_RegisterCallbackReceiveMessage and
// BACnetStack_RegisterCallbackSendMessage no longer exist in the current
// 6.x header - they were renamed/widened to the *ForPort variants below.
// NOTE: none of these extern declarations use `const` on their parameters,
// even though the header's fpCallback* wrappers below do. cgo generates its
// own export header for these Go-side functions WITHOUT `const` qualifiers
// (it does not look at the header at all for its own generated
// declarations), so an extern declaration here that adds `const` back
// creates a signature GCC treats as a conflicting redeclaration and fails
// the build. Keep these bare; the fpCallback* C wrappers immediately below
// are what actually presents the header's real (const-qualified) contract
// to the stack.
extern uint16_t goCallbackReceiveMessageForPort(uint8_t* message, uint16_t maxMessageLength, uint8_t* sourceConnectionString, uint8_t* sourceConnectionStringLength, uint8_t* destinationConnectionString, uint8_t* destinationConnectionStringLength, uint8_t maxConnectionStringLength, uint32_t* networkPortInstance);
extern uint16_t goCallbackSendMessageForPort(uint8_t* message, uint16_t messageLength, uint8_t* connectionString, uint8_t connectionStringLength, uint32_t networkPortInstance, bool broadcast);
extern CASBACnetTime goCallbackGetSystemTime();
extern void goCallbackLogDebugMessage(char* message, uint16_t messageLength, uint8_t messageType);

// Get Property Callbacks
extern bool goCallbackGetPropertyCharString(uint32_t deviceInstance, uint16_t objectType, uint32_t objectInstance, uint32_t propertyIdentifier, char* value, uint32_t* valueElementCount, uint32_t maxElementCount, uint8_t* encodingType, bool useArrayIndex, uint32_t propertyArrayIndex, uint32_t* errorCode);
extern bool goCallbackGetPropertyReal(uint32_t deviceInstance, uint16_t objectType, uint32_t objectInstance, uint32_t propertyIdentifier, float* value, bool useArrayIndex, uint32_t propertyArrayIndex, uint32_t* errorCode);
extern bool goCallbackGetPropertyBool(uint32_t deviceInstance, uint16_t objectType, uint32_t objectInstance, uint32_t propertyIdentifier, bool* value, bool useArrayIndex, uint32_t propertyArrayIndex, uint32_t* errorCode);
extern bool goCallbackGetPropertyEnumerated(uint32_t deviceInstance, uint16_t objectType, uint32_t objectInstance, uint32_t propertyIdentifier, uint32_t* value, bool useArrayIndex, uint32_t propertyArrayIndex, uint32_t* errorCode);
extern bool goCallbackGetPropertyUnsignedInteger(uint32_t deviceInstance, uint16_t objectType, uint32_t objectInstance, uint32_t propertyIdentifier, uint32_t* value, bool useArrayIndex, uint32_t propertyArrayIndex, uint32_t* errorCode);
extern bool goCallbackGetPropertyOctetString(uint32_t deviceInstance, uint16_t objectType, uint32_t objectInstance, uint32_t propertyIdentifier, uint8_t* value, uint32_t* valueElementCount, uint32_t maxElementCount, bool useArrayIndex, uint32_t propertyArrayIndex, uint32_t* errorCode);

// C Callback Declarations
// ====================================================
// DECLARATIONS ONLY here - the DEFINITIONS live in
// cas_bacnet_stack_trampolines.c, not in this comment. cgo compiles this
// preamble comment as source TWICE (once for this file's own translation
// unit, once again inside the generated _cgo_export.c that defines the
// //export shims) - so any actual function BODY placed here is a duplicate
// C symbol and fails to link. A real, separately-compiled .c file is
// compiled exactly once. See cas_bacnet_stack_trampolines.c's own header
// comment for the rest of this explanation and
// cas_bacnet_stack/README.md's "toolchain findings" section.
// General Callbacks
extern uint16_t fpCallbackReceiveMessageForPort(uint8_t* message, const uint16_t maxMessageLength, uint8_t* sourceConnectionString, uint8_t* sourceConnectionStringLength, uint8_t* destinationConnectionString, uint8_t* destinationConnectionStringLength, const uint8_t maxConnectionStringLength, uint32_t* networkPortInstance);
extern uint16_t fpCallbackSendMessageForPort(const uint8_t* message, const uint16_t messageLength, const uint8_t* connectionString, const uint8_t connectionStringLength, const uint32_t networkPortInstance, bool broadcast);
extern CASBACnetTime fpCallbackGetSystemTime();
extern void fpCallbackLogDebugMessage(const char* message, const uint16_t messageLength, const uint8_t messageType);

// Get Property Callbacks
extern bool fpCallbackGetPropertyCharString(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, char* value, uint32_t* valueElementCount, const uint32_t maxElementCount, uint8_t* encodingType, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode);
extern bool fpCallbackGetPropertyReal(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, float* value, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode);
extern bool fpCallbackGetPropertyBool(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, bool* value, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode);
extern bool fpCallbackGetPropertyEnumerated(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, uint32_t* value, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode);
extern bool fpCallbackGetPropertyUnsignedInteger(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, uint32_t* value, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode);
extern bool fpCallbackGetPropertyOctetString(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, uint8_t* value, uint32_t* valueElementCount, const uint32_t maxElementCount, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode);
*/
import "C"

import (
	"unsafe"

	"github.com/chipkin/bacnet-profile-example-b-ss-go/common"
)

// -----------------------------------------------------------------------------
// Registration.
//
// FIX #2 as originally specified ("pass the C trampoline directly, no
// (*[0]byte) cast, so cgo type-checks the registration against the real
// header") does not compile as literally stated on the Go toolchain
// actually available in this repo (go1.26.3): cgo now represents a bare
// reference to a C function (`C.fpCallbackXxx`, used as a value rather than
// called) as an untyped `unsafe.Pointer`-producing expression
// (`_Cgo_ptr(_Cfpvar_fp_fpCallbackXxx)`), and Go's own assignability rules
// do not implicitly convert `unsafe.Pointer` to the named `*[0]byte`
// parameter type `BACnetStack_RegisterCallback*` expects - so EVERY
// registration, not just the ones the original skeleton already cast,
// needs an explicit `(*[0]byte)(unsafe.Pointer(...))` conversion to build
// at all on this toolchain. This is a toolchain-level representation
// change, not a design regression: it was hit and confirmed empirically
// while building this example (see CHANGELOG.md / AGENTS.md "verified
// findings"), not assumed. The real type-safety win from fix #2 survives
// regardless: every fpCallbackXxx C trampoline below is declared with the
// EXACT parameter types copied from the current header (see the `extern`
// block above), so a future header change to any of these callback
// signatures fails this file's OWN C compilation (the fpCallbackXxx
// definition vs. its extern declaration, and - for the transport pair -
// against the header's typedef'd function-pointer parameter when GCC
// elaborates the `#include`) long before it would ever reach a silent
// runtime ABI mismatch.
// -----------------------------------------------------------------------------

// RegisterCallbacks registers every callback this example needs: the
// transport pair (FIX #1: the *ForPort variants), system time (FIX #3: the
// CASBACnetTime signature), debug logging, and all six Get*Property
// callbacks (FIX #4: the original skeleton only wired CharString and Real -
// this example's device model also needs Bool, Enumerated,
// UnsignedInteger and OctetString).
func RegisterCallbacks() {
	C.BACnetStack_RegisterCallbackReceiveMessageForPort((*[0]byte)(unsafe.Pointer(C.fpCallbackReceiveMessageForPort)))
	C.BACnetStack_RegisterCallbackSendMessageForPort((*[0]byte)(unsafe.Pointer(C.fpCallbackSendMessageForPort)))
	C.BACnetStack_RegisterCallbackGetSystemTime((*[0]byte)(unsafe.Pointer(C.fpCallbackGetSystemTime)))
	C.BACnetStack_RegisterCallbackLogDebugMessage((*[0]byte)(unsafe.Pointer(C.fpCallbackLogDebugMessage)))

	C.BACnetStack_RegisterCallbackGetPropertyCharacterString((*[0]byte)(unsafe.Pointer(C.fpCallbackGetPropertyCharString)))
	C.BACnetStack_RegisterCallbackGetPropertyReal((*[0]byte)(unsafe.Pointer(C.fpCallbackGetPropertyReal)))
	C.BACnetStack_RegisterCallbackGetPropertyBool((*[0]byte)(unsafe.Pointer(C.fpCallbackGetPropertyBool)))
	C.BACnetStack_RegisterCallbackGetPropertyEnumerated((*[0]byte)(unsafe.Pointer(C.fpCallbackGetPropertyEnumerated)))
	C.BACnetStack_RegisterCallbackGetPropertyUnsignedInteger((*[0]byte)(unsafe.Pointer(C.fpCallbackGetPropertyUnsignedInteger)))
	C.BACnetStack_RegisterCallbackGetPropertyOctetString((*[0]byte)(unsafe.Pointer(C.fpCallbackGetPropertyOctetString)))
}

// -----------------------------------------------------------------------------
// The shared UDP socket. Package-level, not a closure capture - see the
// package doc comment and common/device_state.go for why.
// -----------------------------------------------------------------------------

var udp common.SimpleUDP

// SetupUDP binds the shared UDP socket. Call once, before RegisterCallbacks.
func SetupUDP(port uint16) error { return udp.Setup(port) }

// ShutdownUDP closes the shared UDP socket.
func ShutdownUDP() { udp.Shutdown() }

// -----------------------------------------------------------------------------
// General callbacks
// -----------------------------------------------------------------------------

//export goCallbackReceiveMessageForPort
func goCallbackReceiveMessageForPort(message *C.uint8_t, maxMessageLength C.uint16_t, sourceConnectionString *C.uint8_t, sourceConnectionStringLength *C.uint8_t, destinationConnectionString *C.uint8_t, destinationConnectionStringLength *C.uint8_t, maxConnectionStringLength C.uint8_t, networkPortInstance *C.uint32_t) C.uint16_t {
	// The out-params arrive UNINITIALIZED - write every one we do not fill
	// with real data, or the stack reads garbage.
	*sourceConnectionStringLength = 0
	*destinationConnectionStringLength = 0
	// FIX #1 (Go side): this is a single-transport (BACnet/IP only) B-SS
	// device, so the resolved port is always this example's one Network
	// Port - write BACNET_NETWORK_PORT_DEFAULT's value via the constant the
	// rest of this example uses (common.NetworkPortInstance), matching the
	// convention documented in the task/adapters README for single-
	// transport builds.
	*networkPortInstance = C.uint32_t(common.NetworkPortInstance)

	if maxConnectionStringLength < 6 {
		return 0 // cannot even fit an IPv4 connection string
	}

	datagram, ok := udp.Recv()
	if !ok {
		return 0 // nothing waiting this tick
	}
	length := len(datagram.Message)
	if length > int(maxMessageLength) {
		return 0 // larger than the stack's receive buffer - drop it
	}
	if length > 0 {
		C.memcpy(unsafe.Pointer(message), unsafe.Pointer(&datagram.Message[0]), C.size_t(length))
	}

	// 6-byte IPv4 connection string: 4 IP octets, then the port BIG-endian.
	srcBuf := (*[6]C.uint8_t)(unsafe.Pointer(sourceConnectionString))
	srcBuf[0] = C.uint8_t(datagram.FromIP[0])
	srcBuf[1] = C.uint8_t(datagram.FromIP[1])
	srcBuf[2] = C.uint8_t(datagram.FromIP[2])
	srcBuf[3] = C.uint8_t(datagram.FromIP[3])
	srcBuf[4] = C.uint8_t(datagram.FromPort >> 8)
	srcBuf[5] = C.uint8_t(datagram.FromPort & 0xFF)
	*sourceConnectionStringLength = 6

	return C.uint16_t(length)
}

//export goCallbackSendMessageForPort
func goCallbackSendMessageForPort(message *C.uint8_t, messageLength C.uint16_t, connectionString *C.uint8_t, connectionStringLength C.uint8_t, networkPortInstance C.uint32_t, broadcast C.bool) C.uint16_t {
	// FIX #1 (Go side): there is only one Network Port in this example, so
	// networkPortInstance is intentionally ignored - nothing to branch on.
	_ = networkPortInstance
	_ = broadcast
	if connectionStringLength < 6 {
		return 0
	}
	octets := (*[6]C.uint8_t)(unsafe.Pointer(connectionString))
	toIP := [4]byte{byte(octets[0]), byte(octets[1]), byte(octets[2]), byte(octets[3])}
	toPort := uint16(octets[4])<<8 | uint16(octets[5])

	buffer := C.GoBytes(unsafe.Pointer(message), C.int(messageLength))
	udp.Send(buffer, toIP, toPort)
	return messageLength
}

//export goCallbackGetSystemTime
func goCallbackGetSystemTime() C.CASBACnetTime {
	return C.CASBACnetTime(nowUnixSeconds())
}

//export goCallbackLogDebugMessage
func goCallbackLogDebugMessage(message *C.char, messageLength C.uint16_t, messageType C.uint8_t) {
	logDebugMessage(C.GoStringN(message, C.int(messageLength)), uint8(messageType))
}

// -----------------------------------------------------------------------------
// Get*Property callbacks
//
// FIX #3: goCallbackGetSystemTime above is declared CASBACnetTime, not
// time_t as the original skeleton had it - the header's
// RegisterCallbackGetSystemTime takes CASBACnetTime (typedef int64_t
// CASBACnetTime;), and cgo now resolves that correctly because the real
// header is #include'd.
//
// FIX #4: Bool/Enumerated/UnsignedInteger/OctetString added below,
// alongside the CharString/Real the skeleton already had, mirroring the
// property-to-callback mapping the Rust sibling used for this same B-SS
// device model (main.rs's get_property_* functions).
//
// GC GOTCHA: for the CharString/OctetString buffers, write directly into
// the C-supplied pointer (never hand a bare Go []byte's backing array to
// C across the call boundary) - see the package doc comment.
// -----------------------------------------------------------------------------

//export goCallbackGetPropertyReal
func goCallbackGetPropertyReal(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.float, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	v, ok := getPropertyReal(uint32(deviceInstance), uint16(objectType), uint32(objectInstance), uint32(propertyIdentifier), bool(useArrayIndex), uint32(propertyArrayIndex), (*uint32)(unsafe.Pointer(errorCode)))
	if !ok {
		return C.bool(false)
	}
	*value = C.float(v)
	return C.bool(true)
}

//export goCallbackGetPropertyBool
func goCallbackGetPropertyBool(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.bool, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	v, ok := getPropertyBool(uint32(deviceInstance), uint16(objectType), uint32(objectInstance), uint32(propertyIdentifier), bool(useArrayIndex), uint32(propertyArrayIndex), (*uint32)(unsafe.Pointer(errorCode)))
	if !ok {
		return C.bool(false)
	}
	*value = C.bool(v)
	return C.bool(true)
}

//export goCallbackGetPropertyEnumerated
func goCallbackGetPropertyEnumerated(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.uint32_t, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	v, ok := getPropertyEnumerated(uint32(deviceInstance), uint16(objectType), uint32(objectInstance), uint32(propertyIdentifier), bool(useArrayIndex), uint32(propertyArrayIndex), (*uint32)(unsafe.Pointer(errorCode)))
	if !ok {
		return C.bool(false)
	}
	*value = C.uint32_t(v)
	return C.bool(true)
}

//export goCallbackGetPropertyUnsignedInteger
func goCallbackGetPropertyUnsignedInteger(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.uint32_t, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	v, ok := getPropertyUnsignedInteger(uint32(deviceInstance), uint16(objectType), uint32(objectInstance), uint32(propertyIdentifier), bool(useArrayIndex), uint32(propertyArrayIndex), (*uint32)(unsafe.Pointer(errorCode)))
	if !ok {
		return C.bool(false)
	}
	*value = C.uint32_t(v)
	return C.bool(true)
}

//export goCallbackGetPropertyCharString
func goCallbackGetPropertyCharString(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.char, valueElementCount *C.uint32_t, maxElementCount C.uint32_t, encodingType *C.uint8_t, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	s, ok := getPropertyCharString(uint32(deviceInstance), uint16(objectType), uint32(objectInstance), uint32(propertyIdentifier), bool(useArrayIndex), uint32(propertyArrayIndex), (*uint32)(unsafe.Pointer(errorCode)))
	if !ok {
		return C.bool(false)
	}
	b := []byte(s)
	if len(b) > int(maxElementCount) {
		b = b[:maxElementCount]
	}
	// Write directly into the C-supplied buffer - never hand C a pointer
	// into a Go-managed slice that outlives this call.
	if len(b) > 0 {
		C.memcpy(unsafe.Pointer(value), unsafe.Pointer(&b[0]), C.size_t(len(b)))
	}
	*valueElementCount = C.uint32_t(len(b))
	*encodingType = C.uint8_t(common.CharacterStringEncodingUTF8)
	return C.bool(true)
}

//export goCallbackGetPropertyOctetString
func goCallbackGetPropertyOctetString(deviceInstance C.uint32_t, objectType C.uint16_t, objectInstance C.uint32_t, propertyIdentifier C.uint32_t, value *C.uint8_t, valueElementCount *C.uint32_t, maxElementCount C.uint32_t, useArrayIndex C.bool, propertyArrayIndex C.uint32_t, errorCode *C.uint32_t) C.bool {
	// This B-SS example does not currently serve any Octet_String property
	// (no object in docs/objects.json needs one) - registered for
	// completeness/parity with the header's full Get*Property set (per
	// task fix #4), but always declines. Add a case here the same way
	// getPropertyCharString.go's siblings do if you add an object that
	// needs one.
	_ = deviceInstance
	_ = objectType
	_ = objectInstance
	_ = propertyIdentifier
	_ = value
	_ = valueElementCount
	_ = maxElementCount
	_ = useArrayIndex
	_ = propertyArrayIndex
	_ = errorCode
	return C.bool(false)
}

// -----------------------------------------------------------------------------
// Thin Go wrappers around the stack's setup/run exports, so main.go never
// needs its own `import "C"` (only this package does).
// -----------------------------------------------------------------------------

func GetAPIMajorVersion() uint32 { return uint32(C.BACnetStack_GetAPIMajorVersion()) }
func GetAPIMinorVersion() uint32 { return uint32(C.BACnetStack_GetAPIMinorVersion()) }
func GetAPIPatchVersion() uint32 { return uint32(C.BACnetStack_GetAPIPatchVersion()) }
func GetAPIBuildVersion() uint32 { return uint32(C.BACnetStack_GetAPIBuildVersion()) }

func AddDevice(deviceInstance uint32) bool {
	return bool(C.BACnetStack_AddDevice(C.uint32_t(deviceInstance)))
}

func SetServiceEnabled(deviceInstance uint32, service uint32, enabled bool) bool {
	return bool(C.BACnetStack_SetServiceEnabled(C.uint32_t(deviceInstance), C.uint32_t(service), C.bool(enabled)))
}

func AddObject(deviceInstance uint32, objectType uint16, objectInstance uint32) bool {
	return bool(C.BACnetStack_AddObject(C.uint32_t(deviceInstance), C.uint16_t(objectType), C.uint32_t(objectInstance)))
}

// AddNetworkPortObject: FIX not needed - already the folded single-call
// signature in the 6.x header (verified against source/CASBACnetStackDLL.h
// directly, cgo picks it up as-is).
func AddNetworkPortObject(deviceInstance, objectInstance uint32, networkType, protocolLevel uint8, networkNumber uint16, networkNumberQuality uint8, referencePort uint32) bool {
	return bool(C.BACnetStack_AddNetworkPortObject(
		C.uint32_t(deviceInstance),
		C.uint32_t(objectInstance),
		C.uint8_t(networkType),
		C.uint8_t(protocolLevel),
		C.uint16_t(networkNumber),
		C.uint8_t(networkNumberQuality),
		C.uint32_t(referencePort),
	))
}

func SetPropertyEnabled(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, enabled bool) bool {
	return bool(C.BACnetStack_SetPropertyEnabled(C.uint32_t(deviceInstance), C.uint16_t(objectType), C.uint32_t(objectInstance), C.uint32_t(propertyIdentifier), C.bool(enabled)))
}

// SendIAm broadcasts an unsolicited I-Am. connectionString must be exactly
// 6 bytes (4 IPv4 octets, then the port big-endian).
func SendIAm(deviceInstance uint32, connectionString [6]byte, networkPortInstance uint32, broadcast bool) bool {
	cConnStr := (*C.uint8_t)(unsafe.Pointer(&connectionString[0]))
	return bool(C.BACnetStack_SendIAm(
		C.uint32_t(deviceInstance),
		cConnStr,
		C.uint8_t(6),
		C.uint32_t(networkPortInstance),
		C.bool(broadcast),
		C.uint16_t(0), // destinationNetwork: local network
		nil,
		C.uint8_t(0),
	))
}

// Tick processes incoming messages and timers. Call it continuously.
func Tick() bool { return bool(C.BACnetStack_Tick()) }
