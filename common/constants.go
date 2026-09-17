// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.

// Package common holds the BACnet enumeration values and device state shared
// between main.go and the cas_bacnet_stack adapter package.
//
// Every value matches the BACnet standard (ANSI/ASHRAE 135) and the CAS
// BACnet Stack enumerations, and every constant name matches the
// C++/C#/Python/Rust editions of this file so the languages diff 1:1.
package common

// -- BACnet object types (Object_Type enumeration) --------------------------
const (
	ObjectTypeAnalogInput     uint16 = 0
	ObjectTypeBinaryInput     uint16 = 3
	ObjectTypeDevice          uint16 = 8
	ObjectTypeMultiStateInput uint16 = 13
	ObjectTypeNetworkPort     uint16 = 56
)

// -- BACnet property identifiers (Property_Identifier enumeration) ----------
const (
	PropertyIdentifierObjectName                 uint32 = 77
	PropertyIdentifierObjectType                 uint32 = 79
	PropertyIdentifierPresentValue                uint32 = 85
	PropertyIdentifierDescription                uint32 = 28
	PropertyIdentifierVendorName                  uint32 = 121
	PropertyIdentifierVendorIdentifier            uint32 = 120
	PropertyIdentifierModelName                   uint32 = 70
	PropertyIdentifierFirmwareRevision            uint32 = 44
	PropertyIdentifierApplicationSoftwareVersion  uint32 = 12
	PropertyIdentifierOutOfService                uint32 = 81
	PropertyIdentifierUnits                       uint32 = 117
	PropertyIdentifierPolarity                    uint32 = 84
	PropertyIdentifierNumberOfStates              uint32 = 74
	PropertyIdentifierStateText                   uint32 = 110
	PropertyIdentifierApduLength                  uint32 = 399
	PropertyIdentifierReferencePort               uint32 = 483
	PropertyIdentifierBacnetIPUDPPort             uint32 = 412
	PropertyIdentifierBacnetIPMode                uint32 = 408
	PropertyIdentifierIPAddress                   uint32 = 400
	PropertyIdentifierIPSubnetMask                uint32 = 411
	PropertyIdentifierIPDefaultGateway            uint32 = 401
)

// -- BACnet engineering units (Engineering_Units enumeration) ---------------
// Full list: submodules/cas-bacnet-stack/source/BACnetEngineeringUnits.h
const EngineeringUnitsDegreesCelsius uint32 = 62

// -- BACnet polarity (Polarity enumeration, for Binary objects) -------------
// Full list: submodules/cas-bacnet-stack/source/BACnetPolarity.h
const PolarityNormal uint32 = 0

// -- BACnet/IP mode (BACnetIPMode enumeration, for the Network Port) --------
// Full list: submodules/cas-bacnet-stack/source/BACnetIPMode.h
const BacnetIPModeNormal uint32 = 0

// -- BACnet services (Services_Supported enumeration) -----------------------
// Used with BACnetStack_SetServiceEnabled(). These are BIT NUMBERS, not
// service-choice values.
const (
	ServiceReadProperty uint32 = 12
	ServiceWhoHas       uint32 = 33
	ServiceWhoIs        uint32 = 34
	ServiceIHave        uint32 = 27
	ServiceIAm          uint32 = 26
)

// -- Network Port object network type (BACnetNetworkType enumeration) -------
const NetworkPortNetworkTypeIPV4 uint8 = 5

// -- Network Port object protocol level (BACnetProtocolLevel enumeration) ---
const NetworkPortProtocolLevelBACnetApplication uint8 = 2

// NetworkPortReferencePortNone: the lowest protocol layer references this
// sentinel instead of another port. Also the default networkPortInstance
// for a single-port device that never calls AddNetworkPortObject
// (BACNET_NETWORK_PORT_DEFAULT).
const NetworkPortReferencePortNone uint32 = 4194303

// -- Network_Number_Quality (BACnetNetworkNumberQuality, cl. 12.56.11) ------
const NetworkNumberQualityUnknown uint8 = 0

// -- Character string encoding (cl. 20.2.9). 0 = UTF-8. ----------------------
const CharacterStringEncodingUTF8 uint8 = 0

// -- BACnet error codes (Error_Code enumeration) -----------------------------
// Full list: submodules/cas-bacnet-stack/source/BACnetErrorCode.h
const ErrorCodeInvalidArrayIndex uint32 = 42

// NetworkPortInstance: this example's own Network Port object instance. Not
// a BACnet enumeration value - just this example's own configuration.
// Defined here (rather than as a main.go-local const) because the transport
// callbacks in cas_bacnet_stack_adapter.go are plain package-level //export
// functions that cannot capture a main()-local variable and need this value
// at compile time to fill the out-param networkPortInstance.
const NetworkPortInstance uint32 = 1 // "Vermilion"

// BACnetNetworkPortDefault: BACNET_NETWORK_PORT_DEFAULT - the sentinel a
// single-transport receive callback should report when it has no better
// per-message answer than "the device's only port".
const BACnetNetworkPortDefault uint32 = 4194303

// -- This example's device/object model --------------------------------------
// See the top-level README.md / docs/objects.json for the full picture.
// "CHANGE ALL OF THIS BEFORE YOU SHIP" if you turn this example into your
// own device.
const (
	DeviceName        = "Rainbow"
	DeviceDescription = "Chipkin CAS BACnet Stack example - B-SS (Smart Sensor) profile. " +
		"Demonstrates DS-RP-B: ReadProperty plus Who-Is/I-Am with read-only sensor objects."
	VendorName             = "Chipkin Automation Systems"
	VendorIdentifier uint32 = 389
	ModelName              = "CAS BACnet Stack Example - B-SS"
	FirmwareRevision       = "1.0.0"
	ApplicationSoftwareVersion = "1.0.0"

	AnalogInputInstance            uint32 = 1 // "Bronze"
	BinaryInputInstance            uint32 = 1 // "Emerald"
	MultiStateInputInstance        uint32 = 1 // "Hot Pink"
	MultiStateInputNumberOfStates  uint32 = 3
	BinaryInput1Value              uint32 = 0 // inactive - the series-wide starting value
	MultiStateInput1Value          uint32 = 1 // state 1 ("On")
	MaxApduLength                  uint32 = 1476 // BACnet/IP APDU length
)

// MultiStateInputStateText: the three named states of Multi-State Input 1
// ("Hot Pink"), 1-indexed per BACnet's State_Text array (index 0 = state 1).
var MultiStateInputStateText = [3]string{"On", "Off", "Auto"}

