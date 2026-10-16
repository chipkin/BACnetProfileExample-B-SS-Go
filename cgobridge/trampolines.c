// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.
//
// trampolines.c - the C halves of the stack's callbacks
// =============================================================================
// WORKAROUND - ADAPTER DEFECT, cas-bacnet-stack#3968 (see TODO.md): the
// stack's Go adapter at the pin does not compile, so this example carries its
// own callback trampolines. Each function below has the exact signature the
// stack's header gives the callback (the adapter's generated
// CASBACnetStackAdapterTypes.h, included in place from the submodule), and forwards to the //export'ed Go function of the same
// name with "go" in place of "fp" (callbacks.go).
//
// They are defined here, in a real .c file that cgo compiles once: cgo also
// copies the preamble of a file with //export functions into _cgo_export.c, so
// a definition there would be a duplicate symbol at link time. The
// cgobridge_register* functions at the end hand them to the stack's
// BACnetStack_RegisterCallback* pointers (the adapter's DLL mode); the C
// compiler checks each against the generated callback type.
//
// _cgo_export.h is the cgo-generated header that declares the //export'ed Go
// functions. cgo does not emit `const` in it, hence the casts on the
// read-only pointers the stack passes in.
// =============================================================================

#include <stdint.h>
#include <stdbool.h>
#include "CASBACnetStackAdapter.h"
#include "_cgo_export.h"

// Transport and time
uint16_t fpCallbackReceiveMessageForPort(uint8_t* message, const uint16_t maxMessageLength, uint8_t* sourceConnectionString, uint8_t* sourceConnectionStringLength, uint8_t* destinationConnectionString, uint8_t* destinationConnectionStringLength, const uint8_t maxConnectionStringLength, uint32_t* networkPortInstance) {
	return goCallbackReceiveMessageForPort(message, maxMessageLength, sourceConnectionString, sourceConnectionStringLength, destinationConnectionString, destinationConnectionStringLength, maxConnectionStringLength, networkPortInstance);
}
uint16_t fpCallbackSendMessageForPort(const uint8_t* message, const uint16_t messageLength, const uint8_t* connectionString, const uint8_t connectionStringLength, const uint32_t networkPortInstance, bool broadcast) {
	return goCallbackSendMessageForPort((uint8_t*)message, messageLength, (uint8_t*)connectionString, connectionStringLength, networkPortInstance, broadcast);
}
CASBACnetTime fpCallbackGetSystemTime() {
	return goCallbackGetSystemTime();
}

// Get*Property
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

// Registration, called from callbacks.go
void cgobridge_registerCommonCallbacks(void) {
	BACnetStack_RegisterCallbackReceiveMessageForPort(fpCallbackReceiveMessageForPort);
	BACnetStack_RegisterCallbackSendMessageForPort(fpCallbackSendMessageForPort);
	BACnetStack_RegisterCallbackGetSystemTime(fpCallbackGetSystemTime);
}
void cgobridge_registerGetPropertyReal(void) { BACnetStack_RegisterCallbackGetPropertyReal(fpCallbackGetPropertyReal); }
void cgobridge_registerGetPropertyEnumerated(void) { BACnetStack_RegisterCallbackGetPropertyEnumerated(fpCallbackGetPropertyEnumerated); }
void cgobridge_registerGetPropertyUnsignedInteger(void) { BACnetStack_RegisterCallbackGetPropertyUnsignedInteger(fpCallbackGetPropertyUnsignedInteger); }
void cgobridge_registerGetPropertyCharString(void) { BACnetStack_RegisterCallbackGetPropertyCharacterString(fpCallbackGetPropertyCharString); }
void cgobridge_registerGetPropertyBool(void) { BACnetStack_RegisterCallbackGetPropertyBool(fpCallbackGetPropertyBool); }
void cgobridge_registerGetPropertyOctetString(void) { BACnetStack_RegisterCallbackGetPropertyOctetString(fpCallbackGetPropertyOctetString); }
