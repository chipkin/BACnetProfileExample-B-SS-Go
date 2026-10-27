# TODO

Open items and known limits at 1.2.0. Nothing here stops the example from
building, running or answering a client as [README.md](README.md) describes. Where
a line has a `ref:` tag, quote it when you contact Chipkin support.

## Known gaps

- **This port still carries its own cgo callback trampolines** instead of using the
  stack's importable Go adapter. See "Adapter defect" below.
- **The Linux build and the release workflow are not verified.** The stack library is
  built with the stack's CMake presets (see the README) and the Windows job combines
  the Visual Studio image with a MinGW-w64 gcc for cgo. This was checked on Windows
  only; `.github/workflows/release.yml` has not been run since it changed to use the
  presets. (ref: chipkin/chipkin-bacnet-testing-tool#455)
- **`common/` sends a frame the stack marks as a broadcast to the one address the stack
  gave, as a unicast.** The shared `common/` helper (the same design is in every
  example of the series) does this, so a router or gateway's reply that cl. 6.4.2
  requires to be a broadcast goes only to the requester. No effect on this sensor has
  been observed. Fix: change `common/`, then copy it into every example.
  (ref: chipkin/chipkin-bacnet-testing-tool#698)
- **The device hears its own broadcasts.** The operating system delivers a datagram
  sent to the subnet broadcast address back to the sending socket, so the start-up
  I-Am (and any Who-Is answer) is received by the device itself. It is harmless on a
  sensor. The stack does not yet treat these as its own transmissions, which on a
  router becomes a broadcast storm. (ref: chipkin/cas-bacnet-stack#4604)

## Not implemented, on purpose

- **No WriteProperty, no COV, no alarming, no scheduling, no trending.** A B-SS Smart
  Sensor is read-only (ANSI/ASHRAE 135-2024 Annex L.4.5), and a WriteProperty gets a
  Reject `unrecognized-service`.
- **The Device's `Object_Name` and instance are set at start-up** (`--deviceName`,
  `--deviceID`), not over the network, and are not kept across a restart. Start the
  example with the same options each time.
- **The sensor readings are simulated.** Replace the Simulation in `main.go` with your
  own sampling to use real hardware (TUTORIAL.md explains how).
- **No `--xml` option** (the C++ example has one).

## VeriBACnet

- A VeriBACnet run of this project (43 BTL tests) is 34 PASS, 0 FAIL, 9 SKIP; see
  `tests/veribacnet/RESULTS.md`. Four of the skips need an operator to restart the
  device with another name or instance, and five do not apply to a device that cannot
  create or delete objects.

## Adapter defect, [cas-bacnet-stack#3968](https://github.com/chipkin/cas-bacnet-stack/issues/3968): the Go adapter did not build at the earlier Stack Pin

**Status at the current Stack Pin (`6.x` @ `69b6249f`):** the upstream issue is
closed. The stack's `adapters/golang/` is now an importable package
(`package casbacnetstack`, with its own `go.mod` and a separate
`CASBACnetStackAdapterTrampolines.c`). This port has not switched to it yet: it
still uses the workaround below, which builds and passes the Wire Tests at the
new pin. Switching is the remaining step of this item.

The defect as it stood:
`submodules/cas-bacnet-stack/adapters/golang/CASBACnetStackAdapter.go` at
`29b83de6` (unchanged since `64b2cfac5`; hand-maintained - `adapters/README.md` says no generator writes the
Go adapter, cgo derives the bindings from the header) cannot be used in place:

1. **It does not compile.** `RegisterBACnetStackCallbacks()` passes each C
   trampoline bare, e.g. line 160
   `C.BACnetStack_RegisterCallbackReceiveMessageForPort(C.fpCallbackReceiveMessageForPort)`
   (all ten registrations, lines 160-171). cgo types a C function used as a
   value as `unsafe.Pointer`, while the generated binding's parameter is
   `*[0]byte`. Compiled in place (as a file of a `package main` test module,
   with `-I` to `source/`, mingw gcc 13.2.0, go1.21.6):

   ```
   .\adapter.go:160:54: cannot use (_Cgo_ptr(_Cfpvar_fp_fpCallbackReceiveMessageForPort)) (value of type unsafe.Pointer) as *[0]byte value in argument to (_Cfunc_BACnetStack_RegisterCallbackReceiveMessageForPort)
   ... (the same for lines 161-171)
   ```

   The comment above those lines (#584/#2111) says the `(*[0]byte)` cast was
   removed on purpose, so cgo type-checks each registration; with this
   toolchain the bare reference does not compile at all.
2. **It is not importable.** It is `package main` (line 6) with its own example
   database (`device`, `analogInput`, the `ExampleDB*` types), no `go.mod`, and
   no `#cgo CFLAGS` include path for the `#include "CASBACnetStackDLL.h"` it
   needs (line 17) - only `#cgo LDFLAGS: -L. -lCASBACnetStack_x64_Release`
   (line 15), relative to the build directory.
3. It does not declare `goCallback*` Go functions for its `extern` declarations;
   a user has to supply all ten `//export` functions in the same package.

**Workaround (port code, marked):** `cgobridge/callbacks.go` and
`cgobridge/trampolines.c` carry the same trampoline pattern (C wrapper ->
`//export`ed Go function), registered in C (`cgobridge_register*` in
`trampolines.c`) through the stack's generated C++ adapter in DLL mode
(`adapters/cpp/`, compiled in place by `cgobridge/stack_adapter.c`), so the C
compiler checks each trampoline against the generated callback type and a
header change still fails the build. **When the adapter builds and is importable, delete
`cgobridge/callbacks.go`'s trampolines and `cgobridge/trampolines.c` and use
it.**

## Smaller notes

- The interactive commands are line-mode (key + Enter, `u`/`d` for the arrows)
  because Go's standard library cannot read raw key presses. TUTORIAL.md
  "Interactive keys are a simplification".
