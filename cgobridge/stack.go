// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.
//
// stack.go - this example's cgo bridge to the CAS BACnet Stack
// =============================================================================
// THE STACK IS LOADED AT RUN TIME, THROUGH ITS OWN ADAPTER, USED IN PLACE.
// stack_adapter.c compiles the stack's generated C++ adapter
// (submodules/cas-bacnet-stack/adapters/cpp/, DLL mode) straight from the
// submodule: every BACnetStack_<Name> is a function-pointer variable that
// LoadBACnetFunctions() binds with LoadLibrary/GetProcAddress (Windows) or
// dlopen/dlsym (Linux). The executable therefore has no load-time dependency
// on CASBACnetStack_x64_Release.dll / libCASBACnetStack_x64_Release.so: when
// the library is missing, main() runs and says so (main.go "Load the CAS
// BACnet Stack"). Nothing from the stack is copied into this repository.
//
// FindStackLibrary() looks for the library in the executable's folder first,
// then, for a development checkout, in submodules/cas-bacnet-stack/bin/ above
// it (where README "Build" builds it).
//
// cgo cannot call a C function-pointer variable from Go, so each stack call
// below goes through a one-line static C function in this preamble.
//
// WORKAROUND - ADAPTER DEFECT (cas-bacnet-stack#3968). The stack's Go adapter
// (adapters/golang/CASBACnetStackAdapter.go) is a `package main` sample that
// does not compile at the pin, so the callback trampolines it would supply are
// in callbacks.go and trampolines.c. See TODO.md.
//
// Every stack call in this example is in this package; common/ stays cgo-free.
// =============================================================================

// Package cgobridge connects this example to the CAS BACnet Stack through cgo.
package cgobridge

/*
#cgo CFLAGS: -I${SRCDIR}/../submodules/cas-bacnet-stack/adapters/cpp -DCAS_BACNET_STACK_LIB_TYPE_DLL
#cgo linux LDFLAGS: -ldl

#include <stdlib.h>
#include "CASBACnetStackAdapter.h"

extern bool cgobridge_load_stack(const char* folder, const char* fileName);

static uint32_t cgobridge_GetAPIMajorVersion(void) { return BACnetStack_GetAPIMajorVersion(); }
static uint32_t cgobridge_GetAPIMinorVersion(void) { return BACnetStack_GetAPIMinorVersion(); }
static uint32_t cgobridge_GetAPIPatchVersion(void) { return BACnetStack_GetAPIPatchVersion(); }
static uint32_t cgobridge_GetAPIBuildVersion(void) { return BACnetStack_GetAPIBuildVersion(); }
static bool cgobridge_AddDevice(uint32_t deviceInstance) { return BACnetStack_AddDevice(deviceInstance); }
static bool cgobridge_SetServiceEnabled(uint32_t deviceInstance, uint32_t service, bool enabled) { return BACnetStack_SetServiceEnabled(deviceInstance, service, enabled); }
static bool cgobridge_AddObject(uint32_t deviceInstance, uint16_t objectType, uint32_t objectInstance) { return BACnetStack_AddObject(deviceInstance, objectType, objectInstance); }
static bool cgobridge_AddNetworkPortObject(uint32_t deviceInstance, uint32_t objectInstance, uint8_t networkType, uint8_t protocolLevel, uint16_t networkNumber, uint8_t networkNumberQuality, uint32_t referencePort) {
	return BACnetStack_AddNetworkPortObject(deviceInstance, objectInstance, networkType, protocolLevel, networkNumber, networkNumberQuality, referencePort);
}
static bool cgobridge_SetPropertyEnabled(uint32_t deviceInstance, uint16_t objectType, uint32_t objectInstance, uint32_t propertyIdentifier, bool enabled) {
	return BACnetStack_SetPropertyEnabled(deviceInstance, objectType, objectInstance, propertyIdentifier, enabled);
}
static void cgobridge_Tick(void) { BACnetStack_Tick(); }
static bool cgobridge_SendIAm(uint32_t deviceInstance, const uint8_t* connectionString, uint8_t connectionStringLength, uint32_t networkPortInstance, bool broadcast) {
	return BACnetStack_SendIAm(deviceInstance, connectionString, connectionStringLength, networkPortInstance, broadcast, 0, NULL, 0);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"
)

// StackLibraryFileName is the CAS BACnet Stack library's file name on this OS.
func StackLibraryFileName() string {
	if runtime.GOOS == "windows" {
		return "CASBACnetStack_x64_Release.dll"
	}
	return "libCASBACnetStack_x64_Release.so"
}

// FindStackLibrary looks for the stack library in the run directory (the
// folder that holds this executable) and then, for a development checkout, in
// submodules/cas-bacnet-stack/bin/ in that folder or one of its parents.
// It returns the folder it was found in ("" if none) and every folder it
// looked in, in order.
func FindStackLibrary() (found string, searched []string) {
	executable, err := os.Executable()
	if err == nil {
		if resolved, err := filepath.EvalSymlinks(executable); err == nil {
			executable = resolved
		}
	}
	var candidates []string
	if err == nil {
		runDirectory := filepath.Dir(executable)
		candidates = append(candidates, runDirectory)
		for directory, level := runDirectory, 0; level < 3; level++ {
			bin := filepath.Join(directory, "submodules", "cas-bacnet-stack", "bin")
			if info, err := os.Stat(bin); err == nil && info.IsDir() {
				candidates = append(candidates, bin)
				break
			}
			parent := filepath.Dir(directory)
			if parent == directory {
				break
			}
			directory = parent
		}
	}
	for _, folder := range candidates {
		searched = append(searched, folder)
		if info, err := os.Stat(filepath.Join(folder, StackLibraryFileName())); err == nil && !info.IsDir() {
			return folder, searched
		}
	}
	return "", searched
}

// LoadStack opens the stack library in folder through the stack's adapter
// (LoadBACnetFunctions: bind every export, check the API version). It must
// succeed before any other function in this package is called.
func LoadStack(folder string) error {
	cFolder := C.CString(folder)
	defer C.free(unsafe.Pointer(cFolder))
	cFileName := C.CString(StackLibraryFileName())
	defer C.free(unsafe.Pointer(cFileName))
	if !bool(C.cgobridge_load_stack(cFolder, cFileName)) {
		return errors.New(C.GoString(C.CASBACnetStackAdapter_LastError()))
	}
	return nil
}

// StackVersion is the loaded stack's version, "major.minor.patch.build".
// The Device's Firmware_Revision and the start-up banner both use it.
func StackVersion() string {
	return fmt.Sprintf("%d.%d.%d.%d",
		uint32(C.cgobridge_GetAPIMajorVersion()), uint32(C.cgobridge_GetAPIMinorVersion()),
		uint32(C.cgobridge_GetAPIPatchVersion()), uint32(C.cgobridge_GetAPIBuildVersion()))
}

// AddDevice is BACnetStack_AddDevice.
func AddDevice(deviceInstance uint32) bool {
	return bool(C.cgobridge_AddDevice(C.uint32_t(deviceInstance)))
}

// SetServiceEnabled is BACnetStack_SetServiceEnabled.
func SetServiceEnabled(deviceInstance uint32, service uint32, enabled bool) bool {
	return bool(C.cgobridge_SetServiceEnabled(C.uint32_t(deviceInstance), C.uint32_t(service), C.bool(enabled)))
}

// AddObject is BACnetStack_AddObject.
func AddObject(deviceInstance uint32, objectType uint16, objectInstance uint32) bool {
	return bool(C.cgobridge_AddObject(C.uint32_t(deviceInstance), C.uint16_t(objectType), C.uint32_t(objectInstance)))
}

// AddNetworkPortObject is BACnetStack_AddNetworkPortObject.
func AddNetworkPortObject(deviceInstance, objectInstance uint32, networkType, protocolLevel uint8,
	networkNumber uint16, networkNumberQuality uint8, referencePort uint32) bool {
	return bool(C.cgobridge_AddNetworkPortObject(
		C.uint32_t(deviceInstance), C.uint32_t(objectInstance),
		C.uint8_t(networkType), C.uint8_t(protocolLevel),
		C.uint16_t(networkNumber), C.uint8_t(networkNumberQuality),
		C.uint32_t(referencePort)))
}

// SetPropertyEnabled is BACnetStack_SetPropertyEnabled.
func SetPropertyEnabled(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, enabled bool) bool {
	return bool(C.cgobridge_SetPropertyEnabled(C.uint32_t(deviceInstance), C.uint16_t(objectType),
		C.uint32_t(objectInstance), C.uint32_t(propertyIdentifier), C.bool(enabled)))
}

// Tick is BACnetStack_Tick: process incoming messages and timers.
func Tick() { C.cgobridge_Tick() }

// sendIAm is BACnetStack_SendIAm to one 6-octet BACnet/IP address on the
// local network (destinationNetwork 0, not the global 0xFFFF).
func sendIAm(deviceInstance uint32, connectionString [6]byte, networkPortInstance uint32, broadcast bool) bool {
	cConnectionString := C.CBytes(connectionString[:])
	defer C.free(cConnectionString)
	return bool(C.cgobridge_SendIAm(C.uint32_t(deviceInstance),
		(*C.uint8_t)(cConnectionString), C.uint8_t(len(connectionString)),
		C.uint32_t(networkPortInstance), C.bool(broadcast)))
}
