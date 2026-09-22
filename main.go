// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see LICENSE.

// BACnet B-SS (Smart Sensor) Example - Go
// =============================================================================
// A minimal, read-only BACnet/IP device built on the CAS BACnet Stack,
// conforming to the B-SS (Smart Sensor) standardized device profile
// (ANSI/ASHRAE 135 Annex L). See README.md for what this is, TUTORIAL.md for
// how to extend it, docs/PICS.md for the conformance statement, and
// AGENTS.md for the ground rules an agent (or a careful human) should follow
// when touching this code.
//
// CHANGE ALL OF THIS BEFORE YOU SHIP: the device instance, name, vendor,
// model and version constants below (in common/constants.go) describe a
// Chipkin tutorial device, not your product.
// =============================================================================

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	bacnet "github.com/chipkin/bacnet-profile-example-b-ss-go/cas_bacnet_stack"
	"github.com/chipkin/bacnet-profile-example-b-ss-go/common"
)

const appName = "BACnet B-SS (Smart Sensor) Example - Go"
const appVersion = "1.0.1"

func printVersion() {
	fmt.Printf("%s v%s (common v%s)\n", appName, appVersion, common.CommonVersion)
	fmt.Printf("CAS BACnet Stack v%d.%d.%d.%d\n",
		bacnet.GetAPIMajorVersion(), bacnet.GetAPIMinorVersion(),
		bacnet.GetAPIPatchVersion(), bacnet.GetAPIBuildVersion())
}

func printHelp() {
	fmt.Println("Commands (type and press Enter):")
	fmt.Println("  h, help  - show this help")
	fmt.Println("  q, quit  - stop the device")
	fmt.Println("  up, u    - nudge Analog Input 1 (Bronze) up by 1.1 C")
	fmt.Println("  down, d  - nudge Analog Input 1 (Bronze) down by 1.1 C")
}

func fail(format string, args ...interface{}) int {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	return 1
}

func run() int {
	port := flag.Uint("port", 47808, "BACnet/IP UDP port")
	deviceID := flag.Uint("deviceID", 389001, "BACnet Device instance (0..4194302)")
	version := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s\n\nUsage: %s [flags]\n\n", appName, os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *version {
		printVersion()
		return 0
	}

	portValue, err := common.ParsePort(fmt.Sprintf("%d", *port))
	if err != nil {
		return fail("%v", err)
	}
	deviceIDValue, err := common.ParseDeviceID(fmt.Sprintf("%d", *deviceID))
	if err != nil {
		return fail("%v", err)
	}

	common.WithState(func(s *common.DeviceState) { s.DeviceInstance = deviceIDValue })

	// --- Load the CAS BACnet Stack + print version --------------------------
	// Unlike a dynamically-loaded adapter (libloading/ctypes/P-Invoke), cgo
	// links the native library at build time via the #cgo LDFLAGS in
	// cas_bacnet_stack/cas_bacnet_stack_adapter.go - if it cannot be found,
	// the OS loader fails when this executable starts, before main() even
	// runs, with its own native error (e.g. "The code execution cannot
	// proceed because CASBACnetStack_x64_Release.dll was not found" on
	// Windows). See README.md "Build the native CAS BACnet Stack library"
	// for where to copy it.
	printVersion()

	// The Device object's Application_Software_Version (12) and
	// Firmware_Revision (44) can't be plain constants: Application_Software_Version
	// must track this example's own real version, and Firmware_Revision must
	// reflect the underlying CAS BACnet Stack's REAL version, read from the
	// stack itself - it names the platform underneath this app, not the app
	// itself. Computed once here (the native library is already known to be
	// linked and working - printVersion() above just called it), using the
	// same 4 getter calls printVersion() uses for the banner, before the
	// socket is bound or any callback is registered.
	common.ApplicationSoftwareVersion = appVersion
	common.FirmwareRevision = fmt.Sprintf("%d.%d.%d.%d",
		bacnet.GetAPIMajorVersion(), bacnet.GetAPIMinorVersion(),
		bacnet.GetAPIPatchVersion(), bacnet.GetAPIBuildVersion())

	// --- Bind the BACnet/IP socket -------------------------------------------
	if err := bacnet.SetupUDP(portValue); err != nil {
		return fail("could not bind UDP port %d: %v (is another BACnet device already running on this machine? Try --port.)", portValue, err)
	}

	// Capture the BACnet/IP addressing the Network Port object will report.
	local := common.GetLocalIPv4()
	common.WithState(func(s *common.DeviceState) {
		s.BacnetIPUDPPort = portValue
		s.IPAddress = local.Address
		s.IPSubnetMask = local.Netmask
	})
	fmt.Printf("FYI: listening on %d.%d.%d.%d:%d (broadcast %d.%d.%d.%d)\n",
		local.Address[0], local.Address[1], local.Address[2], local.Address[3], portValue,
		local.Broadcast[0], local.Broadcast[1], local.Broadcast[2], local.Broadcast[3])

	// --- Register callbacks ---------------------------------------------------
	bacnet.RegisterCallbacks()

	// --- Create the device -----------------------------------------------------
	if !bacnet.AddDevice(deviceIDValue) {
		return fail("Failed to add the Device %d.", deviceIDValue)
	}

	// Enable ReadProperty (DS-RP-B) - the one mandatory service for B-SS. We
	// set it explicitly to make the profile requirement obvious.
	//
	// We deliberately do NOT enable WriteProperty, ReadPropertyMultiple,
	// SubscribeCOV, or any alarm/event service: a Smart Sensor does not
	// require them, so a faithful B-SS example leaves them off.
	if !bacnet.SetServiceEnabled(deviceIDValue, common.ServiceReadProperty, true) {
		return fail("Failed to enable the ReadProperty service.")
	}

	// Discovery: Who-Is/I-Am (DM-DDB-B) and Who-Has/I-Have (DM-DOB-B).
	//
	// These need enabling even though the device already ANSWERS them. The
	// stack's service defaults are whoIs + whoHas + readProperty only - iAm
	// and iHave are left false. Without these calls the device DOES I-Am and
	// I-Have while telling every client it supports neither
	// (Protocol_Services_Supported is emitted verbatim from this bitstring).
	discoveryOK := bacnet.SetServiceEnabled(deviceIDValue, common.ServiceWhoIs, true) &&
		bacnet.SetServiceEnabled(deviceIDValue, common.ServiceIAm, true) &&
		bacnet.SetServiceEnabled(deviceIDValue, common.ServiceWhoHas, true) &&
		bacnet.SetServiceEnabled(deviceIDValue, common.ServiceIHave, true)
	if !discoveryOK {
		return fail("Failed to enable the discovery services (Who-Is/I-Am, Who-Has/I-Have).")
	}

	// --- Add the read-only sensor objects ---------------------------------------
	if !bacnet.AddObject(deviceIDValue, common.ObjectTypeAnalogInput, common.AnalogInputInstance) {
		return fail("Failed to add Analog Input %d (Bronze).", common.AnalogInputInstance)
	}
	if !bacnet.AddObject(deviceIDValue, common.ObjectTypeBinaryInput, common.BinaryInputInstance) {
		return fail("Failed to add Binary Input %d (Emerald).", common.BinaryInputInstance)
	}
	if !bacnet.AddObject(deviceIDValue, common.ObjectTypeMultiStateInput, common.MultiStateInputInstance) {
		return fail("Failed to add Multi-State Input %d (Hot Pink).", common.MultiStateInputInstance)
	}

	// --- Add the Network Port object --------------------------------------------
	// Every BACnet device (Protocol_Revision 17+) must have at least one
	// Network Port object describing the port it talks on. This one is the
	// BACnet/IP application port; it is the lowest layer, so its reference
	// port is "none". networkNumber 0 with quality "unknown" describes a
	// local port that has not learned its network number - the right answer
	// for a device that is not a router and has not been told one.
	if !bacnet.AddNetworkPortObject(
		deviceIDValue,
		common.NetworkPortInstance,
		common.NetworkPortNetworkTypeIPV4,
		common.NetworkPortProtocolLevelBACnetApplication,
		0, // networkNumber: not configured
		common.NetworkNumberQualityUnknown,
		common.NetworkPortReferencePortNone,
	) {
		return fail("Failed to add Network Port 1 (Vermilion).")
	}

	// --- Enable the OPTIONAL properties we choose to expose ---------------------
	// The stack automatically enables an object's REQUIRED properties when
	// the object is added - so Units, Polarity, Number_Of_States,
	// Out_Of_Service, and the Network Port's BACnet/IP addressing are
	// already enabled; our Get* callbacks just supply their values. Only
	// OPTIONAL properties need SetPropertyEnabled. State_Text is optional
	// on a Multi-State Input, so we enable it here.
	//
	// The Device's Description is optional too, and it is an easy one to
	// get wrong: serving it from a Get callback is NOT enough. The stack
	// checks IsPropertyEnabled BEFORE it ever reaches the callbacks, and
	// for an optional property that check falls back to "is it required?"
	// - which is false. So a Description branch in the callback without
	// this enable is DEAD CODE, and the client reads back Error:
	// unknown-property.
	if !bacnet.SetPropertyEnabled(deviceIDValue, common.ObjectTypeDevice, deviceIDValue, common.PropertyIdentifierDescription, true) {
		return fail("Failed to enable Description on the Device object.")
	}
	if !bacnet.SetPropertyEnabled(deviceIDValue, common.ObjectTypeMultiStateInput, common.MultiStateInputInstance, common.PropertyIdentifierStateText, true) {
		return fail("Failed to enable State_Text on Multi-State Input 1 (Hot Pink).")
	}

	// Who-Is is answered automatically. The spec also requires a device to
	// announce itself on start-up, so broadcast an unsolicited I-Am now (to
	// the local subnet broadcast - the Network Port's own network).
	connStr := [6]byte{local.Broadcast[0], local.Broadcast[1], local.Broadcast[2], local.Broadcast[3], byte(portValue >> 8), byte(portValue & 0xFF)}
	if !bacnet.SendIAm(deviceIDValue, connStr, common.NetworkPortInstance, true) {
		return fail("SendIAm failed.")
	}

	fmt.Printf("FYI: Device %d (\"%s\") ready. Vendor ID %d. Type 'h' + Enter for help.\n", deviceIDValue, common.DeviceName, common.VendorIdentifier)

	// --- Graceful Ctrl+C shutdown ------------------------------------------------
	running := int32(1)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		atomic.StoreInt32(&running, 0) // handle it ourselves instead of the process dying mid-tick
	}()

	commands := spawnCommandReader()

	// --- Run the stack -------------------------------------------------------------
	// BACnetStack_Tick() processes incoming messages and timers. Call it
	// continuously, and drain any completed interactive command. The stack
	// is single-threaded by contract - everything that touches it happens
	// on this one loop; the reader goroutine only ever sends completed
	// commands over the channel, never calls into the stack itself.
	for atomic.LoadInt32(&running) != 0 {
		bacnet.Tick()

		select {
		case cmd := <-commands:
			switch cmd {
			case commandHelp:
				printHelp()
			case commandQuit:
				atomic.StoreInt32(&running, 0)
			case commandUp:
				common.WithState(func(s *common.DeviceState) {
					s.AnalogInput1Value += 1.1
					fmt.Printf("Analog Input 1 (Bronze) = %.1f C\n", s.AnalogInput1Value)
				})
			case commandDown:
				common.WithState(func(s *common.DeviceState) {
					s.AnalogInput1Value -= 1.1
					fmt.Printf("Analog Input 1 (Bronze) = %.1f C\n", s.AnalogInput1Value)
				})
			}
		default: // nothing waiting this tick
		}

		time.Sleep(1 * time.Millisecond) // be a good citizen, don't spin the CPU
	}

	bacnet.ShutdownUDP()
	fmt.Println("FYI: stopped.")
	return 0
}

// -----------------------------------------------------------------------------
// Interactive commands are a simplification - see TUTORIAL.md
// -----------------------------------------------------------------------------
// The C++ edition polls raw console keys (including arrow-key escape
// sequences) every millisecond in its own loop. The C#/Python/Rust editions
// each simplified this differently (see their own main files/TUTORIAL.md);
// the Rust edition in particular settled on LINE-buffered commands (type a
// command, press Enter) read from a background thread, specifically to
// avoid a non-stdlib raw-terminal dependency. This Go edition does the same
// for the same reason: Go's stdlib has no portable non-blocking
// single-keypress read either (that needs golang.org/x/term or a similar
// third-party package), and line-buffered commands keep this example
// dependency-free (stdlib only).
//
// A background goroutine blocks on bufio.Scanner.Scan() (there is no
// portable way to make that non-blocking in Go's stdlib either) and
// forwards completed commands to the main loop over a channel, which the
// main loop drains without blocking (select/default) once per tick. This
// also means a background/piped run (the CI smoke test, for instance)
// simply sees stdin hit EOF immediately, the reader goroutine exits
// quietly, and the device keeps running - Ctrl+C still works via the
// handler above.
// -----------------------------------------------------------------------------

type command int

const (
	commandHelp command = iota
	commandQuit
	commandUp
	commandDown
)

func spawnCommandReader() <-chan command {
	ch := make(chan command)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.ToLower(strings.TrimSpace(scanner.Text()))
			var cmd command
			switch line {
			case "h", "help":
				cmd = commandHelp
			case "q", "quit":
				cmd = commandQuit
			case "up", "u":
				cmd = commandUp
			case "down", "d":
				cmd = commandDown
			default:
				continue
			}
			ch <- cmd
		}
	}()
	return ch
}

func main() {
	os.Exit(run())
}
