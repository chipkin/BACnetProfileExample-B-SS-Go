// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.

// device_state.go
// =============================================================================
// The device's mutable state, in one place.
//
// The C++ edition keeps this state in file-level globals next to its
// callbacks. The Go callbacks (property_dispatch.go) are plain top-level
// functions too - the stack reaches them through cgo //export functions in
// cgobridge/callbacks.go, which can only be top-level functions themselves -
// so they read package-level state, never a main()-local variable.
//
// Every piece of device state a callback needs to read (the Analog
// Input's live value, the Network Port's discovered IP addressing, the
// configured device instance and UDP port) lives in this one package-level
// State value, guarded by a Mutex, and every callback locks it and reads
// from it. main.go also writes through this same lock (on start-up, to
// record the resolved IP addressing; in the interactive-command loop, to
// nudge the Analog Input).
//
// A sync.Mutex is correct here (not e.g. an atomic-per-field scheme) because
// the CAS BACnet Stack is single-threaded by contract - nothing in this
// example spawns a goroutine that touches the stack, and the sensor
// Simulation runs on main()'s tick loop, so the lock is never contended by
// stack callbacks. It is there so a future goroutine (a hardware sampler,
// say) can write these fields safely.
// =============================================================================

package common

import "sync"

// DeviceState holds everything a Get*Property callback (or main()) needs to
// read or write after start-up. Fields fixed for the life of the process
// (the colour names, the Multi-State Input's state count) are plain consts
// in main.go instead of living here - only things that are genuinely
// runtime-configurable or mutable belong in this struct.
type DeviceState struct {
	// DeviceInstance is the BACnet Device instance (--deviceID, default 389001).
	DeviceInstance uint32
	// BacnetIPUDPPort is the BACnet/IP UDP port this device is listening on (--port).
	BacnetIPUDPPort uint16

	// The sensor objects' live values. The Get*Property callbacks only ever
	// hand these back; main.go's sensor Simulation (and the interactive
	// up/down commands) are the only things that change them. On a real
	// device your hardware sampling updates them instead.
	//
	// AnalogInput1Value: degrees Celsius, starts at 21.5.
	// BinaryInput1Value: 0 = inactive, 1 = active.
	// MultiStateInput1Value: 1..MultiStateInputNumberOfStates.
	AnalogInput1Value     float32
	BinaryInput1Value     uint32
	MultiStateInput1Value uint32

	// AnalogInput1Reliability is Analog Input 1's Reliability (cl. 12.2.9):
	// ReliabilityNoFaultDetected while the sensor is healthy,
	// ReliabilityOpenLoop while its (simulated) wire is broken. The stack
	// builds the FAULT bit of Status_Flags from this property by itself.
	AnalogInput1Reliability uint32

	// IPAddress/IPSubnetMask/IPDefaultGateway: BACnet/IP addressing the
	// Network Port reports - filled in at start-up from the host's primary
	// interface. The gateway is left unset (0.0.0.0) for this example.
	IPAddress        [4]byte
	IPSubnetMask     [4]byte
	IPDefaultGateway [4]byte
}

var (
	stateMu sync.Mutex
	state   = DeviceState{
		DeviceInstance:          389001,
		BacnetIPUDPPort:         47808,
		AnalogInput1Value:       21.5,
		BinaryInput1Value:       0,
		MultiStateInput1Value:   1,
		AnalogInput1Reliability: ReliabilityNoFaultDetected,
	}
)

// WithState runs fn holding the state lock, for a read or a write. Every
// package-level callback and main.go itself should go through this rather
// than touching `state` directly, so no caller can forget the lock.
func WithState(fn func(s *DeviceState)) {
	stateMu.Lock()
	defer stateMu.Unlock()
	fn(&state)
}
