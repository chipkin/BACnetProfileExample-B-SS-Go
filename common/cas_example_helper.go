// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.

// cas_example_helper.go
// =============================================================================
// Shared plumbing for the BACnet examples - the Go edition of the C++
// examples' common/CASExampleHelper.{h,cpp} (and the C#/Python/Rust
// editions' common/CASExampleHelper.cs / common/cas_example_helper.py /
// common/cas_example_helper.rs).
//
// DEVIATION FROM THE C#/PYTHON/RUST EDITIONS OF THIS FILE: there,
// register_common_callbacks()/send_i_am()/print_version() also live in
// "cas_example_helper" because their adapter bindings are plain function
// wrappers importable from anywhere. Go's cgo has a harder rule: every
// //export'd callback AND every direct C.BACnetStack_* call must live in
// the SAME package that has the `import "C"` preamble (cas_bacnet_stack -
// see that package's README.md). So this file keeps only the CGO-FREE
// pieces (local IPv4 discovery, CLI argument parsing) - the
// callback-registration and SendIAm wrappers live in
// cas_bacnet_stack/cas_bacnet_stack_adapter.go instead, and main.go calls
// them directly.
//
// The stack PULLS datagrams: its receive callback asks for one queued
// datagram per call and the stack never touches the socket. The
// application (SimpleUDP) owns the socket - keep that split in your own
// project.
// =============================================================================

package common

import (
	"fmt"
	"net"
)

// CommonVersion is the version of the vendored common/ helper itself (NOT
// the example's own version). Bump it whenever anything in common/
// changes, and record the change in common/CHANGELOG.md.
const CommonVersion = "1.0.0"

// LocalIPv4 holds the primary interface's address, guessed netmask, and
// derived broadcast address.
type LocalIPv4 struct {
	Address   [4]byte
	Netmask   [4]byte
	Broadcast [4]byte
}

// GetLocalIPv4 returns the primary IPv4 interface's address, a guessed
// netmask, and the derived broadcast address. The Network Port object
// reports these values, and SendIAm targets the derived subnet broadcast.
//
// DEVIATION FROM THE C++/C# EDITIONS (matches the Python/Rust editions' own
// documented deviation): those use OS-specific interface enumeration to
// read the REAL subnet mask. Go's standard library can enumerate
// interfaces (net.Interfaces()), but picking the "primary" one reliably
// without a route-table lookup is itself platform-specific, so - like the
// Rust edition - this opens a UDP socket "connected" to a public address
// (no packet is actually sent - UDP Dial only asks the OS to pick a local
// source address/route) to learn the outbound-interface IP, and ASSUMES a
// /24 (255.255.255.0) netmask, which is correct on most flat home/office/
// lab networks but not on every network. If your subnet is not a /24, pass
// a real IP_Address/IP_Subnet_Mask into the Network Port some other way
// (e.g. read it from net.Interfaces() yourself, or add a --netmask flag)
// rather than trusting this function blindly in production.
func GetLocalIPv4() LocalIPv4 {
	address := [4]byte{127, 0, 0, 1}
	if conn, err := net.Dial("udp4", "8.8.8.8:80"); err == nil {
		if udpAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			if v4 := udpAddr.IP.To4(); v4 != nil {
				copy(address[:], v4)
			}
		}
		_ = conn.Close()
	}

	if address == ([4]byte{127, 0, 0, 1}) || address[0] == 127 {
		return LocalIPv4{
			Address:   [4]byte{127, 0, 0, 1},
			Netmask:   [4]byte{255, 0, 0, 0},
			Broadcast: [4]byte{127, 255, 255, 255},
		}
	}

	netmask := [4]byte{255, 255, 255, 0}
	var broadcast [4]byte
	for i := 0; i < 4; i++ {
		broadcast[i] = address[i] | ^netmask[i]
	}
	return LocalIPv4{Address: address, Netmask: netmask, Broadcast: broadcast}
}

// ParsePort parses --port <n>: an integer 1..65535.
func ParsePort(raw string) (uint16, error) {
	var v int
	if _, err := fmt.Sscanf(raw, "%d", &v); err != nil || v < 1 || v > 65535 || fmt.Sprintf("%d", v) != raw {
		return 0, fmt.Errorf("--port expects an integer 1..65535, got %q", raw)
	}
	return uint16(v), nil
}

// ParseDeviceID parses --deviceID <inst>: an integer 0..4194302. 4194303 is
// the BACnet "unconfigured" sentinel - a real device may not use it.
func ParseDeviceID(raw string) (uint32, error) {
	var v int64
	if _, err := fmt.Sscanf(raw, "%d", &v); err != nil || v < 0 || v > 4194302 || fmt.Sprintf("%d", v) != raw {
		return 0, fmt.Errorf("--deviceID expects an integer 0..4194302, got %q", raw)
	}
	return uint32(v), nil
}
