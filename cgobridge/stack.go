// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.
//
// stack.go - this example's cgo bridge to the CAS BACnet Stack
// =============================================================================
// THE ADAPTER IS USED IN PLACE, FROM THE SUBMODULE. cgo #includes the stack's
// own header, submodules/cas-bacnet-stack/source/CASBACnetStackDLL.h, through
// the -I path below, and generates and type-checks every BACnetStack_* binding
// from it ("Go binds nothing by hand" - adapters/README.md). Nothing in this
// folder is a copy of the stack's adapter or header.
//
// The library is the one built from the submodule into
// submodules/cas-bacnet-stack/bin/ (README "Build"): the linker reads it from
// there (-L below), and at run time the executable loads it from there too
// (Windows: put that folder on PATH; Linux: the rpath below).
//
// WORKAROUND - ADAPTER DEFECT, NOT FILED. The stack's Go adapter at the pin
// (submodules/cas-bacnet-stack/adapters/golang/CASBACnetStackAdapter.go,
// 64b2cfac5) cannot be used as it is:
//   - it is `package main` with its own example database (no importable
//     package, no go.mod) and has no `#cgo CFLAGS` include path for the
//     header it includes;
//   - it does not compile: RegisterBACnetStackCallbacks() passes each C
//     trampoline bare (`C.BACnetStack_RegisterCallbackGetPropertyReal(
//     C.fpCallbackGetPropertyReal)`), which cgo types as unsafe.Pointer, not
//     the *[0]byte the generated binding takes ("cannot use
//     (_Cgo_ptr(_Cfpvar_fp_fpCallbackReceiveMessageForPort)) (value of type
//     unsafe.Pointer) as *[0]byte value", go1.21.6, all ten registrations).
// So this example carries the smallest thing that adapter would have given
// it: the callback trampolines (callbacks.go + trampolines.c - a C wrapper
// and an //export'ed Go function per callback, the pattern the adapter
// uses) and registrations that cast to *[0]byte. Delete callbacks.go and
// trampolines.c and use the adapter's own when it builds. See TODO.md.
//
// Every direct C.BACnetStack_* call in this example is in this package; the
// Go wrappers below are the ones main.go calls. common/ stays cgo-free.
// =============================================================================

// Package cgobridge connects this example to the CAS BACnet Stack through cgo.
package cgobridge

/*
#cgo CFLAGS: -I${SRCDIR}/../submodules/cas-bacnet-stack/source
#cgo LDFLAGS: -L${SRCDIR}/../submodules/cas-bacnet-stack/bin -lCASBACnetStack_x64_Release
#cgo linux LDFLAGS: -Wl,-rpath,${SRCDIR}/../submodules/cas-bacnet-stack/bin -Wl,-rpath,$ORIGIN

#include <stdlib.h>
#include "CASBACnetStackDLL.h"
*/
import "C"

import "fmt"

// StackVersion is the loaded stack's version, "major.minor.patch.build".
// The Device's Firmware_Revision and the start-up banner both use it.
func StackVersion() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		uint32(C.BACnetStack_GetAPIMajorVersion()), uint32(C.BACnetStack_GetAPIMinorVersion()),
		uint32(C.BACnetStack_GetAPIPatchVersion()), uint32(C.BACnetStack_GetAPIBuildVersion()))
}

// AddDevice is BACnetStack_AddDevice.
func AddDevice(deviceInstance uint32) bool {
	return bool(C.BACnetStack_AddDevice(C.uint32_t(deviceInstance)))
}

// SetServiceEnabled is BACnetStack_SetServiceEnabled.
func SetServiceEnabled(deviceInstance uint32, service uint32, enabled bool) bool {
	return bool(C.BACnetStack_SetServiceEnabled(C.uint32_t(deviceInstance), C.uint32_t(service), C.bool(enabled)))
}

// AddObject is BACnetStack_AddObject.
func AddObject(deviceInstance uint32, objectType uint16, objectInstance uint32) bool {
	return bool(C.BACnetStack_AddObject(C.uint32_t(deviceInstance), C.uint16_t(objectType), C.uint32_t(objectInstance)))
}

// AddNetworkPortObject is BACnetStack_AddNetworkPortObject.
func AddNetworkPortObject(deviceInstance, objectInstance uint32, networkType, protocolLevel uint8,
	networkNumber uint16, networkNumberQuality uint8, referencePort uint32) bool {
	return bool(C.BACnetStack_AddNetworkPortObject(
		C.uint32_t(deviceInstance), C.uint32_t(objectInstance),
		C.uint8_t(networkType), C.uint8_t(protocolLevel),
		C.uint16_t(networkNumber), C.uint8_t(networkNumberQuality),
		C.uint32_t(referencePort)))
}

// SetPropertyEnabled is BACnetStack_SetPropertyEnabled.
func SetPropertyEnabled(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, enabled bool) bool {
	return bool(C.BACnetStack_SetPropertyEnabled(C.uint32_t(deviceInstance), C.uint16_t(objectType),
		C.uint32_t(objectInstance), C.uint32_t(propertyIdentifier), C.bool(enabled)))
}

// Tick is BACnetStack_Tick: process incoming messages and timers.
func Tick() { C.BACnetStack_Tick() }

// sendIAm is BACnetStack_SendIAm to one 6-octet BACnet/IP address on the
// local network (destinationNetwork 0, not the global 0xFFFF).
func sendIAm(deviceInstance uint32, connectionString [6]byte, networkPortInstance uint32, broadcast bool) bool {
	cConnectionString := C.CBytes(connectionString[:])
	defer C.free(cConnectionString)
	return bool(C.BACnetStack_SendIAm(C.uint32_t(deviceInstance),
		(*C.uint8_t)(cConnectionString), C.uint8_t(len(connectionString)),
		C.uint32_t(networkPortInstance), C.bool(broadcast),
		C.uint16_t(0), nil, C.uint8_t(0)))
}
