// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.

// Package common holds the BACnet enumeration values and device state shared
// between main.go and the cgobridge package.
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
	PropertyIdentifierPresentValue               uint32 = 85
	PropertyIdentifierDescription                uint32 = 28
	PropertyIdentifierVendorName                 uint32 = 121
	PropertyIdentifierVendorIdentifier           uint32 = 120
	PropertyIdentifierModelName                  uint32 = 70
	PropertyIdentifierFirmwareRevision           uint32 = 44
	PropertyIdentifierApplicationSoftwareVersion uint32 = 12
	PropertyIdentifierOutOfService               uint32 = 81
	PropertyIdentifierUnits                      uint32 = 117
	PropertyIdentifierPolarity                   uint32 = 84
	PropertyIdentifierNumberOfStates             uint32 = 74
	PropertyIdentifierStateText                  uint32 = 110
	PropertyIdentifierApduLength                 uint32 = 399
	PropertyIdentifierReferencePort              uint32 = 483
	PropertyIdentifierBacnetIPUDPPort            uint32 = 412
	PropertyIdentifierBacnetIPMode               uint32 = 408
	PropertyIdentifierIPAddress                  uint32 = 400
	PropertyIdentifierIPSubnetMask               uint32 = 411
	PropertyIdentifierIPDefaultGateway           uint32 = 401
	PropertyIdentifierReliability                uint32 = 103
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

// -- BACnetReliability (cl. 12.2.9 / cl. 21) ----------------------------------
// NO_FAULT_DETECTED while the sensor is healthy; OPEN_LOOP while its
// (simulated) wire is broken. The stack builds the FAULT bit of Status_Flags
// from Reliability by itself (cl. 12.2.7).
const (
	ReliabilityNoFaultDetected uint32 = 0
	ReliabilityOpenLoop        uint32 = 4
)

// -- Character string encoding (cl. 20.2.9). 0 = UTF-8. ----------------------
const CharacterStringEncodingUTF8 uint8 = 0

// -- BACnet error codes (Error_Code enumeration) -----------------------------
// Full list: submodules/cas-bacnet-stack/source/BACnetErrorCode.h
const ErrorCodeInvalidArrayIndex uint32 = 42

// NetworkPortInstance: this example's own Network Port object instance. Not
// a BACnet enumeration value - just this example's own configuration. main.go
// hands it to cgobridge.SetNetworkPortInstance() so the transport callbacks
// and the start-up I-Am name it.
const NetworkPortInstance uint32 = 1 // "Vermilion"

// -- This example's device/object model --------------------------------------
// See the top-level README.md / docs/objects.json for the full picture.
// "CHANGE ALL OF THIS BEFORE YOU SHIP" if you turn this example into your
// own device.
//
// DeviceName is the Device object's Object_Name - the DEFAULT one.
//
// THIS IS THE ONE THAT WILL BITE YOU. Object_Name must be unique across the
// whole BACnet internetwork (cl. 12.1.1.4), so it must be per-unit
// configurable, exactly like the instance. This example takes it at start-up
// with --deviceName <text> (ParseDeviceNameArg(), stored in
// DeviceState.DeviceName by main.go); DeviceName is only what a unit announces
// when nobody configured it. Ship two units without --deviceName and BOTH
// announce the same Object_Name - a spec violation, and a hard BTL failure.
// In your product, configure the name per unit (a serial number, DIP switches,
// a config file, or this argument) and change the default below to something
// that says "not configured yet".
const (
	DeviceName        = "Chipkin Example B-SS"
	DeviceDescription = "Chipkin CAS BACnet Stack example - B-SS (Smart Sensor) profile. " +
		"Demonstrates DS-RP-B: ReadProperty plus Who-Is/I-Am with read-only sensor objects."
	VendorName              = "Chipkin Automation Systems"
	VendorIdentifier uint32 = 389
	ModelName               = "CAS BACnet Stack Example - B-SS"

	AnalogInputInstance           uint32 = 1 // "Bronze"
	BinaryInputInstance           uint32 = 1 // "Emerald"
	MultiStateInputInstance       uint32 = 1 // "Hot Pink"
	MultiStateInputNumberOfStates uint32 = 3
	MaxApduLength                 uint32 = 1476 // BACnet/IP APDU length
)

// MultiStateInputStateText: the three named states of Multi-State Input 1
// ("Hot Pink"), 1-indexed per BACnet's State_Text array (index 0 = state 1).
var MultiStateInputStateText = [3]string{"On", "Off", "Auto"}

// ApplicationSoftwareVersion / FirmwareRevision: the Device object's
// Application_Software_Version (12) and Firmware_Revision (44). These
// cannot be plain compile-time constants: Application_Software_Version must
// track this example's own real version (main.go's appVersion, which
// common cannot import without creating an import cycle), and
// Firmware_Revision must reflect the underlying CAS BACnet Stack's REAL
// version - it names the platform underneath this app, not the app itself -
// which is only known at runtime via the stack's own version-getter API.
// main.go's run() sets both, once, at start-up (see printVersion() and the
// assignment right after it) before the BACnet/IP socket is bound or any
// callback is registered, so there is no concurrent-write hazard.
var (
	ApplicationSoftwareVersion = "0.0.0"
	FirmwareRevision           = "0.0.0.0"
)
