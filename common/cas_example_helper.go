// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.

// cas_example_helper.go
// =============================================================================
// Shared plumbing for the BACnet examples - the Go edition of the C++
// examples' common/CASExampleHelper.{h,cpp}. It follows the C++ helper's
// conventions (common/ 3.4.0 in BACnetProfileExample-B-SS-CPP, plus 3.5.0's
// --deviceName): the same --help / --version text, the same --port /
// --deviceID / --deviceName parsing (an invalid value prints a warning and the
// default is used), and the same one-line RX / TX console summary of every
// BACnet frame.
//
// DEVIATION FROM THE C++ EDITION OF THIS FILE: there, RegisterCommonCallbacks()
// / SendIAm() / PrintVersion() also call into the stack. Go's cgo has a
// harder rule: every //export'd callback AND every direct C.BACnetStack_*
// call must live in the SAME package that has the `import "C"` preamble
// (cgobridge - see cgobridge/callbacks.go). So this file keeps only
// the CGO-FREE pieces, and takes the stack's version as a string where it
// needs to print it.
//
// The stack PULLS datagrams: its receive callback asks for one queued
// datagram per call and the stack never touches the socket. The application
// (SimpleUDP) owns the socket - keep that split in your own project.
// =============================================================================

package common

import (
	"fmt"
	"net"
	"strconv"
)

// CommonVersion is the version of the vendored common/ helper itself (NOT
// the example's own version). Bump it whenever anything in common/
// changes, and record the change in common/CHANGELOG.md.
const CommonVersion = "1.2.0"

// LocalIPv4 holds the primary interface's address, netmask, and derived
// broadcast address.
type LocalIPv4 struct {
	Address   [4]byte
	Netmask   [4]byte
	Broadcast [4]byte
	Found     bool
}

// GetLocalIPv4 returns the primary IPv4 interface's address, its netmask and
// the derived subnet broadcast (ip | ~mask). The Network Port object reports
// these values, and SendIAm and a broadcast the stack hands the send
// callback target the subnet broadcast.
//
// "Primary" is the interface the OS would route a packet to the internet
// through: a UDP socket is "connected" to a public address (no packet is
// sent - UDP Dial only picks a local source address), and the netmask is
// then read from the interface that owns that address (net.Interfaces()).
// With no route at all Found is false, the address and mask are 0.0.0.0 and
// the broadcast is the limited broadcast 255.255.255.255 (the C++ helper's
// fallback).
func GetLocalIPv4() LocalIPv4 {
	// Not found: 0.0.0.0/0.0.0.0, and the limited broadcast 255.255.255.255,
	// as in the C++ helper.
	notFound := LocalIPv4{Broadcast: [4]byte{255, 255, 255, 255}}
	connection, err := net.Dial("udp4", "8.8.8.8:80")
	if err != nil {
		return notFound
	}
	localAddress, isUDP := connection.LocalAddr().(*net.UDPAddr)
	_ = connection.Close()
	if !isUDP || localAddress.IP.To4() == nil || localAddress.IP.IsLoopback() {
		return notFound
	}
	result := LocalIPv4{Found: true}
	copy(result.Address[:], localAddress.IP.To4())
	result.Netmask = [4]byte{255, 255, 255, 0} // used only if no interface owns the address
	if interfaces, err := net.Interfaces(); err == nil {
		for _, networkInterface := range interfaces {
			addresses, err := networkInterface.Addrs()
			if err != nil {
				continue
			}
			for _, address := range addresses {
				network, isNetwork := address.(*net.IPNet)
				if isNetwork && network.IP.To4() != nil && network.IP.To4().Equal(localAddress.IP.To4()) && len(network.Mask) == 4 {
					copy(result.Netmask[:], network.Mask)
				}
			}
		}
	}
	for octetIndex := 0; octetIndex < 4; octetIndex++ {
		result.Broadcast[octetIndex] = result.Address[octetIndex] | ^result.Netmask[octetIndex]
	}
	return result
}

// PrintVersion prints the three version lines every example in the series
// prints: this example, the CAS BACnet Stack it loaded, and this common/.
func PrintVersion(appName, appVersion, stackVersion string) {
	fmt.Printf("%s v%s\n", appName, appVersion)
	fmt.Printf("CAS BACnet Stack version: %s\n", stackVersion)
	fmt.Printf("Common helper (common/) version: %s\n", CommonVersion)
}

// PrintInteractiveCommands prints the interactive command list (no version
// banner). The terminal stays in line mode, so a key takes effect when Enter
// is pressed, and u / d stand in for the C++ edition's up / down arrows - see
// main.go "Interactive commands".
func PrintInteractiveCommands() {
	fmt.Println("Commands (press the key, then Enter):")
	fmt.Println("  h     - show this help (version + commands)")
	fmt.Println("  q     - quit")
	fmt.Println("  u     - increase Analog Input 1 by 1.1 (up)")
	fmt.Println("  d     - decrease Analog Input 1 by 1.1 (down)")
}

// PrintHelp is the 'h' command: the version banner, then the commands.
func PrintHelp(appName, appVersion, stackVersion string) {
	PrintVersion(appName, appVersion, stackVersion)
	PrintInteractiveCommands()
}

// HandleHelpAndVersionArgs prints --help (also -h, /?) or --version and
// returns true when the program should exit (with 0) without starting the
// device. It prints exactly what 1.1.0 printed: --deviceName is not listed
// (see HandleHelpAndVersionArgsWithOptions). Kept for an example that does
// not read --deviceName; this one does, so main.go calls the other.
func HandleHelpAndVersionArgs(arguments []string, appName, appVersion, stackVersion string) bool {
	return HandleHelpAndVersionArgsWithOptions(arguments, appName, appVersion, stackVersion, false)
}

// HandleHelpAndVersionArgsWithOptions is HandleHelpAndVersionArgs with the
// optional help lines: showDeviceNameCLIOption lists --deviceName, for an
// example that reads it (ParseDeviceNameArg; added in 1.2.0, the Go edition
// of the C++ common/ 3.5.0 showDeviceNameCliOption).
func HandleHelpAndVersionArgsWithOptions(arguments []string, appName, appVersion, stackVersion string, showDeviceNameCLIOption bool) bool {
	for _, argument := range arguments {
		switch argument {
		case "--help", "-h", "/?":
			PrintVersion(appName, appVersion, stackVersion)
			fmt.Println()
			fmt.Println("Usage: <executable> [options]")
			fmt.Println()
			fmt.Println("Options:")
			fmt.Println("  --help, -h        Show this help and exit.")
			fmt.Println("  --version         Show version information and exit.")
			fmt.Println("  --deviceID <n>    BACnet device instance (0..4194302). Each example in")
			fmt.Println("                    this series has its own default so several can run on")
			fmt.Println("                    one subnet at once.")
			if showDeviceNameCLIOption {
				fmt.Println("  --deviceName <text>")
				fmt.Printf("                    The Device object's Object_Name (1..%d bytes, no\n", MaxDeviceNameLength)
				fmt.Println("                    control characters). It must be unique across the BACnet")
				fmt.Println("                    internetwork, so give every unit its own. Default: this")
				fmt.Println("                    example's own name.")
			}
			fmt.Println("  --port <n>        BACnet/IP UDP port (1..65535). Default 47808 (0xBAC0).")
			fmt.Println("                    Use a non-default port to avoid clashing with another")
			fmt.Println("                    BACnet device already on 47808 on this host.")
			fmt.Println()
			PrintInteractiveCommands()
			return true
		case "--version":
			PrintVersion(appName, appVersion, stackVersion)
			return true
		}
	}
	return false
}

// parseWholeNumber accepts only a whole token of decimal digits - "12abc",
// "", "-1" and overflow are all rejected, so a typo is reported instead of
// silently becoming another number.
func parseWholeNumber(text string) (int64, bool) {
	if text == "" {
		return 0, false
	}
	for _, character := range text {
		if character < '0' || character > '9' {
			return 0, false
		}
	}
	parsedValue, err := strconv.ParseInt(text, 10, 64)
	return parsedValue, err == nil
}

// ParsePortArg reads --port <n> (1..65535). An invalid value prints a
// warning and the default is used.
func ParsePortArg(arguments []string, defaultPort uint16) uint16 {
	for argumentIndex := 0; argumentIndex+1 < len(arguments); argumentIndex++ {
		if arguments[argumentIndex] == "--port" {
			parsedValue, ok := parseWholeNumber(arguments[argumentIndex+1])
			if ok && parsedValue > 0 && parsedValue <= 65535 {
				return uint16(parsedValue)
			}
			fmt.Printf("Warning: ignoring invalid --port \"%s\" (want 1..65535); using %d.\n",
				arguments[argumentIndex+1], defaultPort)
		}
	}
	return defaultPort
}

// ParseDeviceIDArg reads --deviceID <n> (0..4194302 - 4194303 is the
// "unconfigured" wildcard, not a usable instance). An invalid value prints a
// warning and the default is used.
func ParseDeviceIDArg(arguments []string, defaultDeviceID uint32) uint32 {
	for argumentIndex := 0; argumentIndex+1 < len(arguments); argumentIndex++ {
		if arguments[argumentIndex] == "--deviceID" {
			parsedValue, ok := parseWholeNumber(arguments[argumentIndex+1])
			if ok && parsedValue < 4194303 {
				return uint32(parsedValue)
			}
			fmt.Printf("Warning: ignoring invalid --deviceID \"%s\" (want 0..4194302); using %d.\n",
				arguments[argumentIndex+1], defaultDeviceID)
		}
	}
	return defaultDeviceID
}

// MaxDeviceNameLength is the longest Device Object_Name --deviceName accepts,
// in UTF-8 bytes. cgobridge copies a character string into the stack's
// buffer of MAX_CHARACTER_STRING_SIZE (256 in this build), so a name this
// long is served whole.
const MaxDeviceNameLength = 255

// ParseDeviceNameArg reads --deviceName <text>: the Device object's
// Object_Name. Object_Name must be unique across the BACnet internetwork
// (cl. 12.1.1.4, 12.11), so two units that share a compile-time name are
// non-conformant the moment both are installed; this makes it per-unit
// configurable at start-up, the way --deviceID makes the instance
// configurable. Valid = 1..MaxDeviceNameLength bytes of UTF-8 and no control
// characters (a name must be printable); anything else prints a warning and
// the default is used. The same rules as the C++ edition's
// ParseDeviceNameArg(): every byte >= 0x20 and not 0x7F (the bytes of a UTF-8
// multi-byte character are all >= 0x80).
func ParseDeviceNameArg(arguments []string, defaultDeviceName string) string {
	for argumentIndex := 0; argumentIndex+1 < len(arguments); argumentIndex++ {
		if arguments[argumentIndex] == "--deviceName" {
			name := arguments[argumentIndex+1]
			printable := len(name) > 0 && len(name) <= MaxDeviceNameLength
			for byteIndex := 0; printable && byteIndex < len(name); byteIndex++ {
				printable = name[byteIndex] >= 0x20 && name[byteIndex] != 0x7F
			}
			if printable {
				return name
			}
			fmt.Printf("Warning: ignoring invalid --deviceName (want 1..%d bytes, no control characters); using \"%s\".\n",
				MaxDeviceNameLength, defaultDeviceName)
		}
	}
	return defaultDeviceName
}

// -----------------------------------------------------------------------------
// One-line frame summary for the RX / TX console lines, e.g.
//   TX 25 bytes to 192.168.1.255:47808 (broadcast) (Network Port 1) - Unconfirmed: I-Am
// Same names as the C++ helper (ANSI/ASHRAE 135 cl. 21). The C++ helper also
// names the object and property of a ReadProperty / WriteProperty and offers
// an --xml dump; this edition stops at the service.
// -----------------------------------------------------------------------------

var confirmedServiceNames = []string{
	"AcknowledgeAlarm", "ConfirmedCOVNotification", "ConfirmedEventNotification",
	"GetAlarmSummary", "GetEnrollmentSummary", "SubscribeCOV", "AtomicReadFile",
	"AtomicWriteFile", "AddListElement", "RemoveListElement", "CreateObject",
	"DeleteObject", "ReadProperty", "ReadPropertyConditional", "ReadPropertyMultiple",
	"WriteProperty", "WritePropertyMultiple", "DeviceCommunicationControl",
	"ConfirmedPrivateTransfer", "ConfirmedTextMessage", "ReinitializeDevice",
	"VTOpen", "VTClose", "VTData", "Authenticate", "RequestKey", "ReadRange",
	"LifeSafetyOperation", "SubscribeCOVProperty", "GetEventInformation",
	"SubscribeCOVPropertyMultiple", "ConfirmedCOVNotificationMultiple",
}

var unconfirmedServiceNames = []string{
	"I-Am", "I-Have", "UnconfirmedCOVNotification", "UnconfirmedEventNotification",
	"UnconfirmedPrivateTransfer", "UnconfirmedTextMessage", "TimeSynchronization",
	"Who-Has", "Who-Is", "UTCTimeSynchronization", "WriteGroup",
	"UnconfirmedCOVNotificationMultiple",
}

var rejectReasonNames = []string{
	"other", "buffer-overflow", "inconsistent-parameters",
	"invalid-parameter-data-type", "invalid-tag", "missing-required-parameter",
	"parameter-out-of-range", "too-many-arguments", "undefined-enumeration",
	"unrecognized-service",
}

var abortReasonNames = []string{
	"other", "buffer-overflow", "invalid-apdu-in-this-state",
	"preempted-by-higher-priority-task", "segmentation-not-supported",
	"security-error", "insufficient-security", "window-size-out-of-range",
	"application-exceeded-reply-time", "out-of-resources", "tsm-timeout",
	"apdu-too-long",
}

func namedOrNumber(label string, names []string, number byte, numberLabel string) string {
	if int(number) < len(names) {
		return fmt.Sprintf("%s: %s", label, names[number])
	}
	return fmt.Sprintf("%s: %s=%d", label, numberLabel, number)
}

// SummarizeBacnetFrame describes a BACnet/IP frame (BVLC + NPDU + APDU) in
// one short line; "?" when it is not one.
func SummarizeBacnetFrame(frame []byte) string {
	if len(frame) < 4 || frame[0] != 0x81 {
		return "?"
	}
	bvlcFunction := frame[1]
	offset := 4
	switch bvlcFunction {
	case 0x0A, 0x0B: // Original-Unicast-NPDU, Original-Broadcast-NPDU
	case 0x04: // Forwarded-NPDU carries the original source's 6-octet B/IP address
		offset += 6
	default:
		return fmt.Sprintf("BVLC function=0x%02X", bvlcFunction)
	}
	if offset+2 > len(frame) {
		return "?"
	}
	control := frame[offset+1]
	offset += 2
	hasDestination := control&0x20 != 0
	if hasDestination {
		if offset+3 > len(frame) {
			return "?"
		}
		offset += 3 + int(frame[offset+2])
	}
	if control&0x08 != 0 { // source specifier
		if offset+3 > len(frame) {
			return "?"
		}
		offset += 3 + int(frame[offset+2])
	}
	if hasDestination {
		offset++ // hop count
	}
	if offset >= len(frame) {
		return "?"
	}
	if control&0x80 != 0 {
		return fmt.Sprintf("Network-Layer-Message: type=%d", frame[offset])
	}
	pduType := frame[offset] >> 4
	at := func(index int) (byte, bool) {
		if index < len(frame) {
			return frame[index], true
		}
		return 0, false
	}
	segmented := frame[offset]&0x08 != 0
	switch pduType {
	case 0:
		serviceOffset := offset + 3
		if segmented {
			serviceOffset = offset + 5
		}
		if service, ok := at(serviceOffset); ok {
			return namedOrNumber("ConfirmedRequest", confirmedServiceNames, service, "service")
		}
		return "ConfirmedRequest"
	case 1:
		if service, ok := at(offset + 1); ok {
			return namedOrNumber("Unconfirmed", unconfirmedServiceNames, service, "service")
		}
		return "Unconfirmed"
	case 2:
		if service, ok := at(offset + 2); ok {
			return namedOrNumber("SimpleACK", confirmedServiceNames, service, "service")
		}
		return "SimpleACK"
	case 3:
		serviceOffset := offset + 2
		if segmented {
			serviceOffset = offset + 4
		}
		if service, ok := at(serviceOffset); ok {
			return namedOrNumber("ComplexACK", confirmedServiceNames, service, "service")
		}
		return "ComplexACK"
	case 4:
		return "SegmentACK"
	case 5:
		if service, ok := at(offset + 2); ok {
			return namedOrNumber("Error", confirmedServiceNames, service, "service")
		}
		return "Error"
	case 6:
		if reason, ok := at(offset + 2); ok {
			return namedOrNumber("Reject", rejectReasonNames, reason, "reason")
		}
		return "Reject"
	case 7:
		if reason, ok := at(offset + 2); ok {
			return namedOrNumber("Abort", abortReasonNames, reason, "reason")
		}
		return "Abort"
	}
	return "?"
}
