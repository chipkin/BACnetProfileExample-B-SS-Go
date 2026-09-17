// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.

// property_dispatch.go
// =============================================================================
// The property-to-callback mapping for this B-SS example's device model,
// called from the //export'd goCallbackGetProperty* trampolines in
// cas_bacnet_stack_adapter.go. Mirrors the Rust sibling's main.rs
// get_property_* functions field-for-field (same object/property routing),
// adapted to Go idiom (value, ok) instead of writing through a raw pointer
// directly in these dispatch functions - the trampolines do the unsafe
// pointer write, these functions stay pure Go.
//
// ADDING AN OBJECT? See TUTORIAL.md "Add a second analog input" before you
// do - a half-added object does NOT fail loudly (same caveat as every
// sibling edition of this example).
// =============================================================================

package casbacnetstack

import (
	"fmt"
	"os"
	"time"

	"github.com/chipkin/bacnet-profile-example-b-ss-go/common"
)

// nowUnixSeconds returns the Unix epoch in SECONDS, for
// goCallbackGetSystemTime.
func nowUnixSeconds() int64 { return time.Now().Unix() }

// logDebugMessage forwards the stack's debug log callback to stderr. The
// message type byte is the stack's own severity enumeration
// (source/CASBACnetStackDLL.h's LOG_DEBUG_MESSAGE_TYPE); we don't filter on
// it here, matching the other editions' simplest-possible debug sink.
func logDebugMessage(message string, messageType uint8) {
	fmt.Fprintf(os.Stderr, "[CAS BACnet Stack][%d] %s\n", messageType, message)
}

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
// Units, and the Network Port's BACnet_IP_Mode.
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
				result, ok = common.BinaryInput1Value, true
				return
			}
			if propertyIdentifier == common.PropertyIdentifierPolarity {
				result, ok = common.PolarityNormal, true // required property of a Binary Input
				return
			}
		}
		if objectType == common.ObjectTypeAnalogInput &&
			objectInstance == common.AnalogInputInstance &&
			propertyIdentifier == common.PropertyIdentifierUnits {
			result, ok = common.EngineeringUnitsDegreesCelsius, true
			return
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
				result, ok = common.MultiStateInput1Value, true // valid range is 1..Number_Of_States
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
			// and the spec has a code for it. Without this the client would
			// silently receive an empty string.
			if errorCode != nil {
				*errorCode = common.ErrorCodeInvalidArrayIndex
			}
			return
		}

		// Object_Name - the colour name for each object.
		if propertyIdentifier == common.PropertyIdentifierObjectName {
			switch {
			case objectType == common.ObjectTypeDevice && objectInstance == s.DeviceInstance:
				result, ok = common.DeviceName, true
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
