// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see LICENSE.

// property_dispatch.go - section 3 of main.go: the Get*Property callbacks
// =============================================================================
// The stack calls these when a client reads a property (main.go registers
// them). For each data type the stack uses a separate callback. Each returns
// (value, true) when it recognises the (object, property) pair, and
// (zero, false) otherwise - the Go form of the C++ reference's "fill *value
// and return true". cgobridge/callbacks.go writes the value into the stack's
// buffer.
//
// THE errorCode OUT-PARAMETER. Every callback ends with errorCode *uint32.
// The stack PRESETS it to success (84) before the call, and reads it only if
// you return false. That gives a declining callback two distinct meanings:
//
//   1. return false and LEAVE errorCode ALONE  -> "I have no opinion on this
//      property." The stack falls back to its own handling (see below).
//   2. return false and SET *errorCode         -> "This read fails, with THIS
//      BACnet error." The client gets exactly that Error-PDU.
//
// Option 2 is new (CAS BACnet Stack issue #974). Do not reach for it
// reflexively - option 1 is still the right answer most of the time.
//
// WHAT false-WITHOUT-AN-ERROR-CODE ACTUALLY DOES - the opposite of what most
// people assume. It does NOT reliably produce a BACnet error. The stack
// errors only for the handful of properties it refuses to invent:
// Present_Value, Number_Of_States, Relinquish_Default, Local_Date,
// Local_Time, and a Network Port's APDU_Length (declining one of those reads
// back as Error: read-access-denied). For EVERYTHING ELSE, a false return
// means the stack SILENTLY SUBSTITUTES a default:
//     Object_Name -> the literal string "undefined"
//     Units       -> no-units (95)
//     otherwise   -> a datatype zero-value
//
// AND THAT FALLBACK IS LORE-BEARING, WHICH IS WHY WE DO NOT "FIX" IT HERE.
// The stack relies on the decline-and-fabricate path to answer required
// properties the application is not expected to serve - the Device's
// Max_APDU_Length_Accepted, APDU_Timeout and Number_Of_APDU_Retries among
// them. Name an error on the catch-all return and those required properties
// start failing instead of answering. So: set *errorCode ONLY where THIS
// device knows the read is wrong. There is exactly one such case below
// (State_Text with an out-of-range array index).
//
// ADDING AN OBJECT? READ THIS FIRST. These callbacks are not uniformly strict:
//   - getPropertyReal / getPropertyEnumerated / getPropertyUnsignedInteger
//     match on object type AND INSTANCE. A new instance falls through them.
//   - getPropertyBool matches on TYPE ONLY, so a new instance gets
//     Out_Of_Service for free.
// So a half-added object does NOT fail loudly: its Present_Value errors, but
// its Object_Name reads back as "undefined" and its Units as no-units, with no
// error at all. Add two objects that way and BOTH report Object_Name
// "undefined" - a spec violation that every scan tool shows as healthy. When
// you add an instance, walk EVERY callback below, then read back every
// required property of the new object and DIFF IT against the existing one.
// TUTORIAL.md "Add a second analog input" lists the edits.
//
// Device state lives in common.DeviceState, reached through
// common.WithState() (see common/device_state.go).
// =============================================================================

package main

import "github.com/chipkin/bacnet-profile-example-b-ss-go/common"

// getPropertyReal answers REAL (floating point) - the Analog Input's
// Present_Value.
func getPropertyReal(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, useArrayIndex bool, propertyArrayIndex uint32, errorCode *uint32) (float32, bool) {
	var result float32
	var ok bool
	common.WithState(func(s *common.DeviceState) {
		if deviceInstance == s.DeviceInstance &&
			objectType == common.ObjectTypeAnalogInput &&
			objectInstance == common.AnalogInputInstance &&
			propertyIdentifier == common.PropertyIdentifierPresentValue {
			// ON REAL HARDWARE: return the live sensor reading here, read
			// from a cached variable your hardware updates (as
			// s.AnalogInput1Value is), NOT directly from a slow/blocking
			// device (I2C, SPI, ADC conversion): this callback runs on the
			// BACnetStack_Tick() call, so blocking it delays all BACnet
			// processing. Sample the sensor on a timer/another goroutine
			// and just hand back the latest value from here.
			result = s.AnalogInput1Value
			ok = true
		}
	})
	return result, ok
}

// getPropertyEnumerated answers ENUMERATED - the Binary Input's
// Present_Value (0 = inactive, 1 = active) and Polarity, the Analog Input's
// Units and Reliability, and the Network Port's BACnet_IP_Mode.
func getPropertyEnumerated(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, useArrayIndex bool, propertyArrayIndex uint32, errorCode *uint32) (uint32, bool) {
	var result uint32
	var ok bool
	common.WithState(func(s *common.DeviceState) {
		if deviceInstance != s.DeviceInstance {
			return
		}
		if objectType == common.ObjectTypeBinaryInput && objectInstance == common.BinaryInputInstance {
			if propertyIdentifier == common.PropertyIdentifierPresentValue {
				// ON REAL HARDWARE: return your cached input state here -
				// the same rule as getPropertyReal above applies.
				result, ok = s.BinaryInput1Value, true
				return
			}
			if propertyIdentifier == common.PropertyIdentifierPolarity {
				result, ok = common.PolarityNormal, true // required property of a Binary Input
				return
			}
		}
		if objectType == common.ObjectTypeAnalogInput && objectInstance == common.AnalogInputInstance {
			if propertyIdentifier == common.PropertyIdentifierUnits {
				result, ok = common.EngineeringUnitsDegreesCelsius, true
				return
			}
			// Reliability (enabled in main.go). The stack also reads it
			// through this callback to build Status_Flags, so once enabled it
			// must always answer: if it declines, the stack silently
			// substitutes 0 (no-fault-detected), and the sensor reports
			// itself healthy forever.
			if propertyIdentifier == common.PropertyIdentifierReliability {
				result, ok = s.AnalogInput1Reliability, true
				return
			}
		}
		if objectType == common.ObjectTypeNetworkPort &&
			objectInstance == common.NetworkPortInstance &&
			propertyIdentifier == common.PropertyIdentifierBacnetIPMode {
			result, ok = common.BacnetIPModeNormal, true // not foreign-device, not BBMD
			return
		}
	})
	return result, ok
}

// getPropertyUnsignedInteger answers UNSIGNED INTEGER - the Multi-State
// Input's Present_Value and Number_Of_States (and its State_Text array
// length), the Device's Vendor_Identifier (the stack also uses
// Vendor_Identifier to build I-Am), and the Network Port's
// Max_APDU_Length_Accepted / Reference_Port / BACnet_IP_UDP_Port.
func getPropertyUnsignedInteger(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, useArrayIndex bool, propertyArrayIndex uint32, errorCode *uint32) (uint32, bool) {
	var result uint32
	var ok bool
	common.WithState(func(s *common.DeviceState) {
		if deviceInstance != s.DeviceInstance {
			return
		}
		if objectType == common.ObjectTypeMultiStateInput && objectInstance == common.MultiStateInputInstance {
			if propertyIdentifier == common.PropertyIdentifierPresentValue {
				result, ok = s.MultiStateInput1Value, true // valid range is 1..Number_Of_States
				return
			}
			if propertyIdentifier == common.PropertyIdentifierNumberOfStates {
				result, ok = common.MultiStateInputNumberOfStates, true // required property
				return
			}
			// State_Text is an array. The stack asks for its LENGTH here
			// (array index 0) before reading each element via
			// getPropertyCharString.
			if propertyIdentifier == common.PropertyIdentifierStateText && useArrayIndex && propertyArrayIndex == 0 {
				result, ok = common.MultiStateInputNumberOfStates, true
				return
			}
		}
		if objectType == common.ObjectTypeDevice &&
			objectInstance == s.DeviceInstance &&
			propertyIdentifier == common.PropertyIdentifierVendorIdentifier {
			result, ok = common.VendorIdentifier, true
			return
		}
		if objectType == common.ObjectTypeNetworkPort && objectInstance == common.NetworkPortInstance {
			if propertyIdentifier == common.PropertyIdentifierApduLength {
				result, ok = common.MaxApduLength, true
				return
			}
			if propertyIdentifier == common.PropertyIdentifierReferencePort {
				result, ok = common.NetworkPortReferencePortNone, true
				return
			}
			if propertyIdentifier == common.PropertyIdentifierBacnetIPUDPPort {
				result, ok = uint32(s.BacnetIPUDPPort), true
				return
			}
		}
	})
	return result, ok
}

// getPropertyBool answers BOOLEAN - Out_Of_Service is a required property
// of every input object and of the Network Port. This is a read-only
// sensor, so nothing is ever out of service: always false.
func getPropertyBool(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, useArrayIndex bool, propertyArrayIndex uint32, errorCode *uint32) (bool, bool) {
	var result bool
	var ok bool
	common.WithState(func(s *common.DeviceState) {
		if deviceInstance != s.DeviceInstance {
			return
		}
		if propertyIdentifier == common.PropertyIdentifierOutOfService &&
			(objectType == common.ObjectTypeAnalogInput ||
				objectType == common.ObjectTypeBinaryInput ||
				objectType == common.ObjectTypeMultiStateInput ||
				objectType == common.ObjectTypeNetworkPort) {
			result, ok = false, true
		}
	})
	return result, ok
}

// getPropertyOctetString answers OCTET STRING - the Network Port's
// BACnet/IP addressing. The stack cannot know the host's IP, so the
// application must supply IP_Address and IP_Subnet_Mask (and
// IP_Default_Gateway). Each is four octets. The stack also reads IP_Address
// (with BACnet_IP_UDP_Port) to build the port's six-octet MAC_Address - so if
// this callback declines, all four read back as empty octet strings, with no
// error.
func getPropertyOctetString(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, useArrayIndex bool, propertyArrayIndex uint32, errorCode *uint32) ([]byte, bool) {
	var result []byte
	var ok bool
	common.WithState(func(s *common.DeviceState) {
		if deviceInstance != s.DeviceInstance ||
			objectType != common.ObjectTypeNetworkPort ||
			objectInstance != common.NetworkPortInstance {
			return
		}
		switch propertyIdentifier {
		case common.PropertyIdentifierIPAddress:
			result, ok = append([]byte(nil), s.IPAddress[:]...), true
		case common.PropertyIdentifierIPSubnetMask:
			result, ok = append([]byte(nil), s.IPSubnetMask[:]...), true
		case common.PropertyIdentifierIPDefaultGateway:
			result, ok = append([]byte(nil), s.IPDefaultGateway[:]...), true
		}
	})
	return result, ok
}

// getPropertyCharString answers CHARACTER STRING - Object_Name for each
// object, the Multi-State Input's State_Text array, and the Device's
// remaining identity strings (Description, Vendor_Name, Model_Name,
// Firmware_Revision, Application_Software_Version).
func getPropertyCharString(deviceInstance uint32, objectType uint16, objectInstance uint32, propertyIdentifier uint32, useArrayIndex bool, propertyArrayIndex uint32, errorCode *uint32) (string, bool) {
	var result string
	var ok bool
	common.WithState(func(s *common.DeviceState) {
		if deviceInstance != s.DeviceInstance {
			return
		}

		// State_Text (optional) - one label per state of the Multi-State
		// Input. It is a BACnet array, so the stack asks for one element at
		// a time by index (1..Number_Of_States). Present_Value 1 -> "On",
		// 2 -> "Off", 3 -> "Auto".
		if objectType == common.ObjectTypeMultiStateInput &&
			objectInstance == common.MultiStateInputInstance &&
			propertyIdentifier == common.PropertyIdentifierStateText &&
			useArrayIndex {
			if propertyArrayIndex >= 1 && propertyArrayIndex <= common.MultiStateInputNumberOfStates {
				result, ok = common.MultiStateInputStateText[propertyArrayIndex-1], true
				return
			}
			// The one place in this file where naming an error is clearly
			// right: the client asked for State_Text[n] and this object has
			// no element n. That is not "no opinion" - it is a wrong read,
			// and the spec has a code for it. (At this Stack Pin the stack
			// also checks n against the length getPropertyUnsignedInteger
			// serves for index 0 and answers invalid-array-index before it
			// gets here; this line is what still answers correctly if that
			// length and this table ever disagree - without it the client
			// would silently receive an empty string.)
			if errorCode != nil {
				*errorCode = common.ErrorCodeInvalidArrayIndex
			}
			return
		}

		// Object_Name - the colour name for each object.
		if propertyIdentifier == common.PropertyIdentifierObjectName {
			switch {
			case objectType == common.ObjectTypeDevice && objectInstance == s.DeviceInstance:
				// The configured name (--deviceName), not the constant: it
				// must be unique per unit. Who-Has by name resolves through
				// this callback too.
				result, ok = s.DeviceName, true
			case objectType == common.ObjectTypeAnalogInput && objectInstance == common.AnalogInputInstance:
				result, ok = "Bronze", true
			case objectType == common.ObjectTypeBinaryInput && objectInstance == common.BinaryInputInstance:
				result, ok = "Emerald", true
			case objectType == common.ObjectTypeMultiStateInput && objectInstance == common.MultiStateInputInstance:
				result, ok = "Hot Pink", true
			case objectType == common.ObjectTypeNetworkPort && objectInstance == common.NetworkPortInstance:
				result, ok = "Vermilion", true
			}
			if ok {
				return
			}
		}

		// The remaining strings are all on the Device object - its
		// identity, read by clients and used to populate the device's
		// I-Am / object list.
		if objectType == common.ObjectTypeDevice && objectInstance == s.DeviceInstance {
			switch propertyIdentifier {
			case common.PropertyIdentifierDescription:
				result, ok = common.DeviceDescription, true
			case common.PropertyIdentifierVendorName:
				result, ok = common.VendorName, true
			case common.PropertyIdentifierModelName:
				result, ok = common.ModelName, true
			case common.PropertyIdentifierFirmwareRevision:
				result, ok = common.FirmwareRevision, true
			case common.PropertyIdentifierApplicationSoftwareVersion:
				result, ok = common.ApplicationSoftwareVersion, true
			}
		}
	})
	return result, ok
}
