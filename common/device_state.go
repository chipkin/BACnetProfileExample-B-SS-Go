// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.

// device_state.go
// =============================================================================
// The device's mutable state, in one place.
//
// THIS IS THE SINGLE MOST IMPORTANT DESIGN POINT IN THIS EXAMPLE. Go's cgo
// //export functions (the Get*Property callbacks in
// cas_bacnet_stack/cas_bacnet_stack_adapter.go) must be top-level,
// non-method, non-closure functions - cgo flatly refuses to export anything
// else. So a callback CANNOT close over any main()-local variable the way
// the C++ edition's lambdas or the Python edition's closures do; it can only
// read package-level state.
//
// So every piece of device state a callback needs to read (the Analog
// Input's live value, the Network Port's discovered IP addressing, the
// configured device instance and UDP port) lives in this one package-level
// State value, guarded by a Mutex, and every callback locks it and reads
// from it. main.go also writes through this same lock (on start-up, to
// record the resolved IP addressing; in the interactive-command loop, to
// nudge the Analog Input).
//
// A sync.Mutex is correct here (not e.g. an atomic-per-field scheme) because
// the CAS BACnet Stack is single-threaded by contract - nothing in this
// example spawns a goroutine that touches the stack, so the lock is never
// contended by stack callbacks. It exists because the interactive-command
// reader goroutine and main()'s tick loop both touch the same fields.
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

	// AnalogInput1Value is Analog Input 1's live present value (degrees
	// Celsius). Starts at 21.5 and is nudged by the interactive up/down
	// commands. A real sensor would update this from hardware instead.
	AnalogInput1Value float32

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
		DeviceInstance:    389001,
		BacnetIPUDPPort:   47808,
		AnalogInput1Value: 21.5,
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
