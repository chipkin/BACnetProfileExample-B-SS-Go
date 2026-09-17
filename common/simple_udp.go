// SPDX-License-Identifier: CC0-1.0
// Public-domain example code (CC0) - see ../LICENSE.

// simple_udp.go
// =============================================================================
// A minimal UDP wrapper for the BACnet examples - the Go edition of the C++
// examples' common/SimpleUDP.{h,cpp} (and the C#/Python/Rust editions'
// common/SimpleUDP.cs / common/simple_udp.py / common/simple_udp.rs), with
// the same responsibilities:
//
//   - own ONE datagram socket bound to the BACnet/IP port,
//   - let the stack PULL inbound datagrams one at a time (Recv()) from its
//     receive callback (the stack never touches the socket - the
//     application does),
//   - send outbound datagrams where the stack's send callback points.
//
// NO TRANSPORT HELPER EXISTS IN THE GO ADAPTER DIRECTORY (adapters/golang/
// in the submodule ships CASBACnetStackAdapter.go and propertybufferhelper/
// only - no UDP helper) - this is written from scratch against Go's stdlib
// net.ListenUDP/net.UDPConn, matching the shape/responsibilities of the
// C#/Python/Rust siblings above.
//
// Poll-based, not queue-based: Recv() calls the OS socket's ReadFromUDP
// directly, with a short read deadline so it never blocks the caller for
// long, once per stack tick. That keeps this whole type usable from a
// single-threaded caller, matching the other editions' synchronous model -
// there must not be a background goroutine touching this socket (the CAS
// BACnet Stack is single-threaded by contract - see AGENTS.md).
//
// This type never sees a BACnet "connection string" - it deals in
// host-order ip/port pairs. Packing the 6-byte connection string (4 IP
// octets + 2 port bytes, port BIG-endian) is cas_example_helper's job,
// exactly as in the other editions.
// =============================================================================

package common

import (
	"fmt"
	"net"
	"os"
	"time"
)

// ReceivedDatagram is one received datagram, handed to the stack's receive callback.
type ReceivedDatagram struct {
	Message  []byte
	FromIP   [4]byte
	FromPort uint16
}

// SimpleUDP is an application-owned, non-blocking (via a short read
// deadline) UDP socket.
type SimpleUDP struct {
	conn *net.UDPConn
}

// Setup binds the socket. Returns an error on bind failure (for example:
// another BACnet device already owns the port exclusively).
func (s *SimpleUDP) Setup(port uint16) error {
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: int(port)})
	if err != nil {
		return err
	}
	s.conn = conn
	return nil
}

// Recv returns the next queued inbound datagram, or ok=false if none arrived
// within a short poll window. Non-blocking in effect: it uses a tiny read
// deadline (rather than SetReadDeadline(time.Now()), which some platforms
// treat as "expired immediately, don't even try") so BACnetStack_Tick() is
// never meaningfully delayed by a socket read - this type is polled once
// per tick from main.go's own loop, not from a background goroutine.
func (s *SimpleUDP) Recv() (ReceivedDatagram, bool) {
	if s.conn == nil {
		return ReceivedDatagram{}, false
	}
	// 2048 bytes: comfortably larger than the BACnet/IP APDU max (1497).
	buffer := make([]byte, 2048)
	_ = s.conn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))
	n, addr, err := s.conn.ReadFromUDP(buffer)
	if err != nil {
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return ReceivedDatagram{}, false // nothing waiting this tick
		}
		fmt.Fprintf(os.Stderr, "Error: UDP receive failed: %v\n", err)
		return ReceivedDatagram{}, false
	}
	var fromIP [4]byte
	copy(fromIP[:], addr.IP.To4())
	return ReceivedDatagram{
		Message:  buffer[:n],
		FromIP:   fromIP,
		FromPort: uint16(addr.Port),
	}, true
}

// Send sends one datagram. Fire-and-forget by design: UDP gives no delivery
// guarantee anyway, so a send error is logged, not propagated - same
// behaviour as the other editions' SimpleUDP.send.
func (s *SimpleUDP) Send(message []byte, toIP [4]byte, toPort uint16) {
	if s.conn == nil {
		return
	}
	addr := &net.UDPAddr{IP: net.IPv4(toIP[0], toIP[1], toIP[2], toIP[3]), Port: int(toPort)}
	if _, err := s.conn.WriteToUDP(message, addr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: UDP send to %v:%d failed: %v\n", addr.IP, toPort, err)
	}
}

// Shutdown closes the socket (best-effort).
func (s *SimpleUDP) Shutdown() {
	if s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
}
