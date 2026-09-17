// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.
//
// cas_bacnet_stack_trampolines.c
// =============================================================================
// The C-side halves of the three-layer cgo callback trampoline pattern
// (C forward declaration in the .go file's cgo preamble -> C wrapper
// defined HERE, matching the CAS BACnet Stack header's exact callback
// signature -> the Go //export function in cas_bacnet_stack_adapter.go /
// property_dispatch.go), as a REAL, separately-compiled translation unit
// rather than function bodies inside the cgo preamble comment.
//
// WHY THIS FILE EXISTS (a toolchain finding made while building this
// example, not a stylistic choice - see CHANGELOG.md / AGENTS.md): cgo
// compiles a .go file's preamble comment as C source TWICE - once for that
// file's own object, and again inside the cgo-generated `_cgo_export.c`
// (which needs the preamble's types/declarations in scope to define the
// `//export` shims). A non-`static` function DEFINITION placed directly in
// the preamble therefore becomes two conflicting definitions of the same
// symbol at link time ("multiple definition of `fpCallbackReceiveMessageForPort`").
// Marking the wrappers `static` avoids that duplicate-symbol error but then
// breaks the OTHER direction: `cas_bacnet_stack_adapter.go`'s
// `RegisterCallbacks()` takes each wrapper's address (`C.fpCallbackXxx`) as
// a value to register with the stack, and that reference is emitted into
// yet another translation unit which then cannot see a `static` symbol
// defined in a different one. A real .c file, compiled exactly once by the
// normal cgo/C build (any `.c` file next to a `.go` file with `import "C"`
// is picked up automatically - no `#cgo` line needed), has neither problem:
// one definition, ordinary external linkage, visible everywhere it is
// declared `extern` (see the preamble's "C Callback Declarations" section).
//
// #include "_cgo_export.h" is the cgo-generated header declaring every
// `//export`ed Go function (goCallbackReceiveMessageForPort, ...) with C
// linkage and C-side integer/bool types - it is what lets a plain .c file
// call into Go-implemented callbacks at all.
// =============================================================================

#include <stdint.h>
#include <stdbool.h>
#include "CASBACnetStackDLL.h" // for the CASBACnetTime typedef
#include "_cgo_export.h"

// General Callbacks
uint16_t fpCallbackReceiveMessageForPort(uint8_t* message, const uint16_t maxMessageLength, uint8_t* sourceConnectionString, uint8_t* sourceConnectionStringLength, uint8_t* destinationConnectionString, uint8_t* destinationConnectionStringLength, const uint8_t maxConnectionStringLength, uint32_t* networkPortInstance) {
	return goCallbackReceiveMessageForPort(message, maxMessageLength, sourceConnectionString, sourceConnectionStringLength, destinationConnectionString, destinationConnectionStringLength, maxConnectionStringLength, networkPortInstance);
}
uint16_t fpCallbackSendMessageForPort(const uint8_t* message, const uint16_t messageLength, const uint8_t* connectionString, const uint8_t connectionStringLength, const uint32_t networkPortInstance, bool broadcast) {
	// The header declares these `const` (the stack promises not to mutate
	// them); goCallbackSendMessageForPort's cgo-generated export signature
	// never has `const` on its pointer params (cgo does not emit `const` in
	// its generated headers), so the cast below is just bridging that
	// (harmless - the Go side only reads these bytes) - not a type-safety
	// hole introduced by this vendored copy.
	return goCallbackSendMessageForPort((uint8_t*)message, messageLength, (uint8_t*)connectionString, connectionStringLength, networkPortInstance, broadcast);
}
CASBACnetTime fpCallbackGetSystemTime() {
	return goCallbackGetSystemTime();
}
void fpCallbackLogDebugMessage(const char* message, const uint16_t messageLength, const uint8_t messageType) {
	goCallbackLogDebugMessage((char*)message, messageLength, messageType);
}

// Get Property Callbacks
bool fpCallbackGetPropertyCharString(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, char* value, uint32_t* valueElementCount, const uint32_t maxElementCount, uint8_t* encodingType, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode) {
	return goCallbackGetPropertyCharString(deviceInstance, objectType, objectInstance, propertyIdentifier, value, valueElementCount, maxElementCount, encodingType, useArrayIndex, propertyArrayIndex, errorCode);
}
bool fpCallbackGetPropertyReal(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, float* value, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode) {
	return goCallbackGetPropertyReal(deviceInstance, objectType, objectInstance, propertyIdentifier, value, useArrayIndex, propertyArrayIndex, errorCode);
}
bool fpCallbackGetPropertyBool(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, bool* value, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode) {
	return goCallbackGetPropertyBool(deviceInstance, objectType, objectInstance, propertyIdentifier, value, useArrayIndex, propertyArrayIndex, errorCode);
}
bool fpCallbackGetPropertyEnumerated(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, uint32_t* value, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode) {
	return goCallbackGetPropertyEnumerated(deviceInstance, objectType, objectInstance, propertyIdentifier, value, useArrayIndex, propertyArrayIndex, errorCode);
}
bool fpCallbackGetPropertyUnsignedInteger(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, uint32_t* value, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode) {
	return goCallbackGetPropertyUnsignedInteger(deviceInstance, objectType, objectInstance, propertyIdentifier, value, useArrayIndex, propertyArrayIndex, errorCode);
}
bool fpCallbackGetPropertyOctetString(const uint32_t deviceInstance, const uint16_t objectType, const uint32_t objectInstance, const uint32_t propertyIdentifier, uint8_t* value, uint32_t* valueElementCount, const uint32_t maxElementCount, const bool useArrayIndex, const uint32_t propertyArrayIndex, uint32_t* errorCode) {
	return goCallbackGetPropertyOctetString(deviceInstance, objectType, objectInstance, propertyIdentifier, value, valueElementCount, maxElementCount, useArrayIndex, propertyArrayIndex, errorCode);
}
