# BACnet B-SS (Smart Sensor) - Go example

A minimal, read-only BACnet/IP device built on the [CAS BACnet
Stack](https://www.chipkin.com/cas-bacnet-stack/), implementing the **B-SS
(BACnet Smart Sensor)** standardized device profile in **Go**, via **cgo**
bindings to the stack's native C library.

This is the fifth language port in a series that also has
[C++](../BACnetProfileExample-B-SS-CPP), [C#](../BACnetProfileExample-B-SS-CS),
[Node.js](../BACnetProfileExample-B-SS-Node),
[Python](../BACnetProfileExample-B-SS-Python) and
[Rust](../BACnetProfileExample-B-SS-Rust) editions, all implementing the
*same* device/object model so the languages diff 1:1.

## What is the B-SS (Smart Sensor) profile?

B-SS (ANSI/ASHRAE 135 Annex L.8, "Smart Sensor") is the standardized device
profile for a simple, read-only sensing device: it presents one or more
Input objects (Analog/Binary/Multi-State), answers ReadProperty for every
required property, and is discoverable via Who-Is/I-Am and Who-Has/I-Have.
It does **not** support WriteProperty, ReadPropertyMultiple, COV, alarms, or
scheduling - those belong to richer profiles (B-AAC, B-ASC, ...) elsewhere
in this series.

## The device this example creates

| Object | Instance | Name | Notes |
|---|:---:|---|---|
| Device | 389001 | Chipkin Example B-SS | vendor ID 389 (Chipkin), `--deviceID` configurable |
| Analog Input | 1 | Bronze | REAL, degrees Celsius, starts 21.5, nudged +/-1.1 via interactive commands |
| Binary Input | 1 | Emerald | starts inactive |
| Multi-State Input | 1 | Hot Pink | state 1 of 3 ("On"/"Off"/"Auto"), `State_Text` enabled |
| Network Port | 1 | Vermilion | BACnet/IP, IPv4 auto-discovered |

See [docs/objects.json](docs/objects.json) for the full property-level
breakdown and [docs/PICS.md](docs/PICS.md) for the formal conformance
statement.

## What this example supports

### BIBBs (BACnet Interoperability Building Blocks)

- **DS-RP-B** - Data Sharing - ReadProperty - B
- **DM-DDB-B** - Device Management - Dynamic Device Binding - B (Who-Is/I-Am)
- **DM-DOB-B** - Device Management - Dynamic Object Binding - B (Who-Has/I-Have)

### Services (executed / B-side)

`SERVICE_READ_PROPERTY`, `SERVICE_WHO_IS`, `SERVICE_I_AM`,
`SERVICE_WHO_HAS`, `SERVICE_I_HAVE`. Nothing else is enabled - no
WriteProperty, no ReadPropertyMultiple, no COV, no alarm/event service.

### Object types

Device, Analog Input, Binary Input, Multi-State Input, Network Port. Every
required property of every object is served; the Device's `Description`
and the Multi-State Input's `State_Text` are the only *optional* properties
this example turns on.

## Requires the CAS BACnet Stack (licensed product)

This example links against the CAS BACnet Stack's native library
(`CASBACnetStack_x64_Release.dll` on Windows, `libCASBACnetStack_x64_Release.so`
on Linux), built from the `submodules/cas-bacnet-stack` submodule (branch
`6.x`, a **licensed** Chipkin product - see that submodule's own license).
This repository ships **no** native binary in git history for anyone who
has not already cloned this submodule with credentials; see "Build the
native CAS BACnet Stack library" below.

## What's in this repository

```
main.go                              CLI, device/object setup, run loop, interactive commands
go.mod

cas_bacnet_stack/                    the vendored + FIXED cgo adapter (see its own README.md)
  cas_bacnet_stack_adapter.go        cgo preamble, callback registration, Get*Property callbacks, thin stack wrappers
  cas_bacnet_stack_trampolines.c     the C-side callback trampolines (see file header for why this is a real .c file)
  README.md                          every fix made to the vendored adapter, and why

common/                              cgo-free shared plumbing (device state, transport, CLI helpers)
  constants.go                       BACnet enumeration values + this example's device/object constants
  device_state.go                    the device's mutable state (package-level, mutex-guarded)
  simple_udp.go                      application-owned, non-blocking UDP socket
  cas_example_helper.go              local IPv4 discovery, CLI arg parsing
  CHANGELOG.md

docs/
  objects.json                       machine-readable object/property model (ported from the C++/Rust editions)
  PICS.md                            BACnet Protocol Implementation Conformance Statement

lib/                                 the built native library lands here (gitignored except for a placeholder)
submodules/cas-bacnet-stack/         the CAS BACnet Stack, branch 6.x (submodule)

.github/workflows/release.yml        CI: build native lib + Go binary, smoke test, release on tag
TUTORIAL.md                          how to extend this example
CHANGELOG.md                         this example's own version history
AGENTS.md                            ground rules for an agent (or a careful human) touching this code
LICENSE                              CC0-1.0 (this example's own code; the submodule has its own license)
```

## Prerequisites

- **Go 1.20+** (this repo was built and verified against `go1.26.3`;
  `go.mod` declares `go 1.20` as the minimum - see `cas_bacnet_stack/README.md`
  for why that floor, matching the vendored adapter's own
  `propertybufferhelper/go.mod` precedent).
- **`CGO_ENABLED=1`** - this example is cgo-only; `CGO_ENABLED=0` will not
  build it. Go turns this on by default when a C compiler is found on
  `PATH`, but CI sets it explicitly (see `.github/workflows/release.yml`).
- **A C compiler for cgo.** On Windows this means **MinGW-w64/gcc** - *not*
  the MSVC toolchain that builds the native DLL itself. They are two
  separate compilers for two separate jobs: MSBuild/MSVC builds
  `CASBACnetStack_x64_Release.dll` from the submodule's C++ sources; cgo's
  gcc compiles this repo's own small C trampoline file and links against
  that already-built DLL. On Linux, any gcc/clang works for both jobs.
- Git, for the submodule.

## Build

### 1. Add the submodule

```sh
git submodule update --init --recursive
```

(Already wired via `.gitmodules` if you cloned this repository directly.)

### 2. Build the native CAS BACnet Stack library

**Reuse, don't rebuild, when possible.** A from-scratch MSVC build of the
native library takes several minutes; if a sibling example in this series
(the Rust or Python edition, for instance) has already built
`CASBACnetStack_x64_Release.dll` at the **same submodule commit** this
repository is pinned to, just copy that file into `lib/` instead of
rebuilding - `git -C submodules/cas-bacnet-stack rev-parse HEAD` from both
repositories tells you whether the commits match.

**Windows (MSBuild, if you do need to build it):**

```powershell
msbuild submodules\cas-bacnet-stack\projects\msvs\BuildCASBACnetStack.sln `
  /p:Configuration=ReleaseDll /p:Platform=x64 /p:PlatformToolset=v143
copy submodules\cas-bacnet-stack\<build output path>\CASBACnetStack_x64_Release.dll lib\
```

**Linux (g++):**

```sh
g++ -shared -fPIC -o lib/libCASBACnetStack_x64_Release.so \
  submodules/cas-bacnet-stack/source/*.cpp -I submodules/cas-bacnet-stack/source
```

### Why only a Release build

Same reasoning as every sibling edition of this example: this repository
only ever builds/ships the **Release** configuration of the native library.
There is no reason for a tutorial example to link a Debug build, and the
vendored adapter skeleton's original `#cgo LDFLAGS: ... -lCASBACnetStack_x64_Debug`
was one of the bugs this fork fixes - see `cas_bacnet_stack/README.md` fix #5.

### 3. Build the Go binary

```sh
CGO_ENABLED=1 go build -o bacnet-b-ss-go .
```

(On Windows PowerShell: `$env:CGO_ENABLED=1; go build -o bacnet-b-ss-go.exe .`)

### 4. Put the native library where the built binary can find it

cgo links `CASBACnetStack_x64_Release` at **build time** via `#cgo LDFLAGS`
pointing at `lib/` (see `cas_bacnet_stack/cas_bacnet_stack_adapter.go`), but
the OS's dynamic loader still needs to **find** the DLL/so at **run time**.
Copy it next to the built executable:

```sh
cp lib/CASBACnetStack_x64_Release.dll .        # Windows, next to bacnet-b-ss-go.exe
cp lib/libCASBACnetStack_x64_Release.so .      # Linux, next to bacnet-b-ss-go
```

If the library cannot be found, the OS loader fails before `main()` runs,
with its own native error (e.g. Windows: *"The code execution cannot
proceed because CASBACnetStack_x64_Release.dll was not found"*).

## Run

```sh
./bacnet-b-ss-go --port 47808 --deviceID 389001
```

### Command-line options

| Flag | Default | Meaning |
|---|---|---|
| `--port` | 47808 | BACnet/IP UDP port |
| `--deviceID` | 389001 | BACnet Device instance (0..4194302) |
| `--version` | - | print version and exit |
| `--help` | - | print flag usage and exit (Go's `flag` package default) |

### Interactive commands

Type a command and press Enter (see "Interactive commands are a
simplification" in `main.go` / `TUTORIAL.md` for why this is
line-buffered, not raw single-keypress, input):

| Command | Effect |
|---|---|
| `h`, `help` | show help |
| `q`, `quit` | stop the device |
| `up`, `u` | nudge Analog Input 1 (Bronze) up by 1.1 C |
| `down`, `d` | nudge Analog Input 1 (Bronze) down by 1.1 C |

Ctrl+C also stops the device cleanly.

## Verify

From another machine (or the same one) on the same subnet, use a BACnet
explorer (e.g. YABE, VTS, or Chipkin's own BACnet Explorer) to send a
Who-Is and confirm Device 389001 ("Chipkin Example B-SS") answers with an I-Am, then
browse its four objects and read their properties.

## How the Go binding differs from the C++/C#/Python/Rust editions

- **cgo, not P/Invoke/ctypes/dlopen/libloading.** The native library is
  linked at **build time** (`#cgo LDFLAGS`), not resolved dynamically at
  run time the way C#'s P/Invoke, Python's `ctypes`, or the Rust edition's
  `libloading::Library` do. The trade-off: no "library not found" runtime
  panic to catch (the OS loader fails before `main()` even starts - see
  "Put the native library where the built binary can find it" above), but
  a rebuild is required if you ever want to point at a different native
  library file name.
- **Package-level state, not closures - same root cause as Rust, different
  mechanism.** Go's cgo `//export` functions must be top-level,
  non-method, non-closure functions - a hard cgo rule, not a style choice.
  So exactly like the Rust edition's plain `extern "C" fn` items, this
  edition's callbacks read/write `common`'s package-level `DeviceState`
  through a mutex (`common.WithState`) rather than capturing a
  `main()`-local variable. See `common/device_state.go`'s doc comment.
- **A real vendored, fixed C adapter file - the only edition in this
  series with one.** The C++ edition talks to the stack's C++ classes
  directly; C#/Python/Rust vendor pure FFI *binding* layers (function
  signatures only) that never shipped broken. The Go adapter is different:
  `submodules/cas-bacnet-stack/adapters/golang/CASBACnetStackAdapter.go` is
  a real, hand-maintained cgo skeleton that had genuinely **stopped
  compiling** against the current 6.x header (renamed transport callbacks,
  a wrong `time_t` vs `CASBACnetTime` type, and only 2 of 6 needed
  `Get*Property` callbacks wired). Fixing it - in this repository's own
  vendored copy only, never in `submodules/` - is real "fix the language
  adapter" work; see `cas_bacnet_stack/README.md` for the complete list.
- **A dependency-free stdlib transport, like Rust's.** No transport helper
  ships in `adapters/golang/` any more than it does in `adapters/rust/`;
  `common/simple_udp.go` is written from scratch against
  `net.ListenUDP`/`net.UDPConn`, matching `simple_udp.rs`'s shape and
  responsibilities.
- **Line-buffered interactive commands, like Rust's.** Go's standard
  library has no portable non-blocking single-keypress read either (that
  needs `golang.org/x/term` or similar) - this edition made the same
  dependency-free choice the Rust edition did, for the same reason.

## The BACnet profile example series

<!-- PROFILE-TABLE:BEGIN (generated from cas-bacnet-stack-examples/docs/profile-table.md - do not edit here) -->
The CAS BACnet Stack supports every standardized device profile in ASHRAE 135-2024 Annex L, and there is one example repository per profile. Pick the profile your device claims, then the language you build in. "Ask" means the example hasn't been built yet for that language - [contact Chipkin](https://store.chipkin.com/contact-us) if you need one.

### Controllers (Annex L.4)

| Profile | C++ | Node.js | C# | Rust | Python | Go |
|---|---|---|---|---|---|---|
| **B-SS** Smart Sensor | [B-SS-CPP](https://github.com/chipkin/BACnetProfileExample-B-SS-CPP) | [B-SS-Node](https://github.com/chipkin/BACnetProfileExample-B-SS-Node) | [B-SS-CS](https://github.com/chipkin/BACnetProfileExample-B-SS-CS) | [B-SS-Rust](https://github.com/chipkin/BACnetProfileExample-B-SS-Rust) | [B-SS-Python](https://github.com/chipkin/BACnetProfileExample-B-SS-Python) | [B-SS-Go](https://github.com/chipkin/BACnetProfileExample-B-SS-Go) |
| **B-SA** Smart Actuator | [B-SA-CPP](https://github.com/chipkin/BACnetProfileExample-B-SA-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-ASC** Application Specific Controller | [B-ASC-CPP](https://github.com/chipkin/BACnetProfileExample-B-ASC-CPP) | [B-ASC-Node](https://github.com/chipkin/BACnetProfileExample-B-ASC-Node) | Ask | Ask | Ask | Ask |
| **B-AAC** Advanced Application Controller | [B-AAC-CPP](https://github.com/chipkin/BACnetProfileExample-B-AAC-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-BC** Building Controller | [B-BC-CPP](https://github.com/chipkin/BACnetProfileExample-B-BC-CPP) | Ask | Ask | Ask | Ask | Ask |

### Life safety controllers (Annex L.5)

| Profile | C++ | Node.js | C# | Rust | Python | Go |
|---|---|---|---|---|---|---|
| **B-LSC** Life Safety Controller | [B-LSC-CPP](https://github.com/chipkin/BACnetProfileExample-B-LSC-CPP) 🚧 | Ask | Ask | Ask | Ask | Ask |
| **B-ALSC** Advanced Life Safety Controller | [B-ALSC-CPP](https://github.com/chipkin/BACnetProfileExample-B-ALSC-CPP) | Ask | Ask | Ask | Ask | Ask |

### Access control controllers (Annex L.6)

| Profile | C++ | Node.js | C# | Rust | Python | Go |
|---|---|---|---|---|---|---|
| **B-ACC** Access Control Controller | [B-ACC-CPP](https://github.com/chipkin/BACnetProfileExample-B-ACC-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-AACC** Advanced Access Control Controller | [B-AACC-CPP](https://github.com/chipkin/BACnetProfileExample-B-AACC-CPP) | Ask | Ask | Ask | Ask | Ask |

### Lighting controllers (Annex L.11)

| Profile | C++ | Node.js | C# | Rust | Python | Go |
|---|---|---|---|---|---|---|
| **B-LD** Lighting Device | [B-LD-CPP](https://github.com/chipkin/BACnetProfileExample-B-LD-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-LS** Lighting Supervisor | [B-LS-CPP](https://github.com/chipkin/BACnetProfileExample-B-LS-CPP) | Ask | Ask | Ask | Ask | Ask |

### Elevator controllers (Annex L.13)

| Profile | C++ | Node.js | C# | Rust | Python | Go |
|---|---|---|---|---|---|---|
| **B-EM** Elevator Monitor | [B-EM-CPP](https://github.com/chipkin/BACnetProfileExample-B-EM-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-EC** Elevator Controller | [B-EC-CPP](https://github.com/chipkin/BACnetProfileExample-B-EC-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-AEC** Advanced Elevator Controller | [B-AEC-CPP](https://github.com/chipkin/BACnetProfileExample-B-AEC-CPP) | Ask | Ask | Ask | Ask | Ask |

### Authentication and authorization (Annex L.14)

| Profile | C++ | Node.js | C# | Rust | Python | Go |
|---|---|---|---|---|---|---|
| **B-AS** Authorization Server | [B-AS-CPP](https://github.com/chipkin/BACnetProfileExample-B-AS-CPP) | Ask | Ask | Ask | Ask | Ask |

### Miscellaneous (Annex L.7, combinable with any one family)

| Profile | C++ | Node.js | C# | Rust | Python | Go |
|---|---|---|---|---|---|---|
| **B-BBMD** Broadcast Management Device | [B-BBMD-CPP](https://github.com/chipkin/BACnetProfileExample-B-BBMD-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-ACDC** Access Control Door Controller | [B-ACDC-CPP](https://github.com/chipkin/BACnetProfileExample-B-ACDC-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-ACCR** Access Control Credential Reader | [B-ACCR-CPP](https://github.com/chipkin/BACnetProfileExample-B-ACCR-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-RTR** Router | [B-RTR-CPP](https://github.com/chipkin/BACnetProfileExample-B-RTR-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-GW** Gateway | [B-GW-CPP](https://github.com/chipkin/BACnetProfileExample-B-GW-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-DAP** Device Address Proxy | [B-DAP-CPP](https://github.com/chipkin/BACnetProfileExample-B-DAP-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-SCHUB** BACnet/SC Hub | [B-SCHUB-CPP](https://github.com/chipkin/BACnetProfileExample-B-SCHUB-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-GENERAL** General device (Annex L.8) | *(satisfied by every example above)* | — | — | — | — | — |

### Operator interfaces and workstations (Annex L.1–L.3, L.9–L.10, L.12)

Client-side profiles.

| Profile | C++ | Node.js | C# | Rust | Python | Go |
|---|---|---|---|---|---|---|
| **B-OD** Operator Display | [B-OD-CPP](https://github.com/chipkin/BACnetProfileExample-B-OD-CPP) | Ask | Ask | Ask | Ask | Ask |
| **B-OWS** Operator Workstation | planned | — | — | — | — | — |
| **B-AWS** Advanced Operator Workstation | planned | — | — | — | — | — |
| **B-XAWS** Extended Advanced Operator Workstation | planned | — | — | — | — | — |
| **B-LSAP** Life Safety Annunciator Panel | planned | — | — | — | — | — |
| **B-LSWS** Life Safety Workstation | planned | — | — | — | — | — |
| **B-ALSWS** Advanced Life Safety Workstation | planned | — | — | — | — | — |
| **B-ACSD** Access Control Security Display | planned | — | — | — | — | — |
| **B-ACWS** Access Control Workstation | planned | — | — | — | — | — |
| **B-AACWS** Advanced Access Control Workstation | planned | — | — | — | — | — |
| **B-LOD** Lighting Operator Display | planned | — | — | — | — | — |
| **B-ALWS** Advanced Lighting Workstation | planned | — | — | — | — | — |
| **B-LCS** Lighting Control Station | planned | — | — | — | — | — |
| **B-ALCS** Advanced Lighting Control Station | planned | — | — | — | — | — |
| **B-ED** Elevator Display | planned | — | — | — | — | — |
| **B-EWS** Elevator Workstation | planned | — | — | — | — | — |
| **B-AEWS** Advanced Elevator Workstation | planned | — | — | — | — | — |

🚧 = in progress. "Ask" = not yet built for that language; contact Chipkin if you need it. Profile definitions: ANSI/ASHRAE 135-2024 Annex L. BIBB definitions: Annex K. Get the stack: <https://store.chipkin.com/services/stacks/bacnet-stack>.
<!-- PROFILE-TABLE:END -->

## References

- ANSI/ASHRAE Standard 135-2024, Annex A (PICS template), Annex K (BIBBs),
  Annex L (device profiles), Clause 12 (object types).
- [CAS BACnet Stack](https://www.chipkin.com/cas-bacnet-stack/) documentation.
- [TUTORIAL.md](TUTORIAL.md) - how to extend this example.
- [docs/PICS.md](docs/PICS.md) - the formal conformance statement.
- [cas_bacnet_stack/README.md](cas_bacnet_stack/README.md) - the vendored
  adapter fixes.
