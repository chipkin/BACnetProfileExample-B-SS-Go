# BACnet B-SS (Smart Sensor) - Go example

A minimal, copy-paste-friendly example showing how to implement the BACnet
**B-SS (Smart Sensor)** device profile in **Go**, using the
[CAS BACnet Stack](https://store.chipkin.com/services/stacks/bacnet-stack)
through **cgo**. It listens on **BACnet/IP (UDP 47808)**, answers
**ReadProperty**, and is discoverable via **Who-Is / I-Am**. Its readings follow
a sensor **Simulation** (labelled below), and it reports a broken sensor the
BACnet way.

This is a faithful Go port of the reworked
[BACnetProfileExample-B-SS-CPP](https://github.com/chipkin/BACnetProfileExample-B-SS-CPP)
(v1.3.0): the same Device, objects, properties, Simulation, Failure Modes,
test-only hook and command-line options, proven by the same
[Wire Tests](#wire-tests), which cannot tell the two apart on the wire.

- **[TUTORIAL.md](TUTORIAL.md)** - how to extend this example and how to review
  it for conformance. Read it when you start turning this into your own device.
- **[docs/PICS.md](docs/PICS.md)** - the Protocol Implementation Conformance
  Statement: every object, every property, and who answers it.

> **Versions:** this document describes **example v1.1.0**, built and verified
> against **CAS BACnet Stack 6.0.23** (`6.x` @ `64b2cfac5`, the `ReleaseDll`
> shared library built from the submodule, its header included in place by
> cgo), at **Protocol_Revision 30**, with the Go `common/` helper at **v1.1.0**
> and `wiretest/` at **0.7.0**. Running the example prints all three versions -
> if what it prints disagrees with this line, trust the program and check
> `CHANGELOG.md`.

## What is the B-SS (Smart Sensor) profile?

**B-SS (BACnet Smart Sensor)**, defined in Annex L of ANSI/ASHRAE 135, is the
simplest BACnet device profile - the standard describes it as *"a simple
sensing device with very limited resources."* It is meant for inexpensive,
fixed-function sensors (temperature, humidity, occupancy, a contact, ...) that
mostly just need to **report what they measure** when asked.

A B-SS device answers **ReadProperty** and is discoverable. It does not have
to support **WriteProperty** (a smart sensor is read-only), **alarming /
event reporting**, **scheduling**, or **trending**, and this example
implements none of them on purpose.

**But it is still a full BACnet device.** Even the simplest profile must
present the standard object model - a **Device** object, a **Network Port**
object (every device needs one), and its sensor objects - and each object
must expose all of its **required properties**. The CAS BACnet Stack
generates most of those automatically (Object_Identifier, Object_Type,
Status_Flags, Object_List, Protocol_*, ...); this example supplies the
handful that are application-specific. The result is conformant for
**Protocol_Revision 30**. [docs/PICS.md](docs/PICS.md) lists every property
and who answers it.

## The device this example creates

```
Device 389001  "Chipkin Example B-SS"   (Vendor 389 - Chipkin Automation Systems)
    │
    ├── Analog Input 1       "Bronze"      Present_Value  21.5 -> ramps 18.0 .. 26.0  (REAL, degrees Celsius)
    ├── Binary Input 1       "Emerald"     Present_Value  toggles every 3 s     (0 = inactive / 1 = active)
    ├── Multi-State Input 1  "Hot Pink"    Present_Value  1 -> 2 -> 3 every 4 s (state, 1..3)
    └── Network Port 1       "Vermilion"   the BACnet/IP port     (required on every device)
```

## The scenario: a sensor a controller reads

What a B-SS is for, in one sentence, from Annex L.4.5 of ANSI/ASHRAE 135-2024:
*"Ability to provide values for any of its BACnet objects upon request."* A
controller or workstation (the **Peer** - any BACnet client, or the Python
client in [`tests/`](tests/README.md)) reads the sensor and gets what it is
measuring **now**:

1. The Peer finds the sensor with **Who-Is** and reads its `Object_List`.
2. It reads Analog Input 1 `Present_Value` - say `22.0` - and again a few
   seconds later - `23.5`. The reading follows the (simulated) room temperature,
   0.5 C a second. Binary Input 1 and Multi-State Input 1 change too.
3. **Failure Mode - a broken sensor.** When the sensor wire breaks, Analog
   Input 1 `Reliability` reads `open-loop` (4) and the **FAULT** bit of
   `Status_Flags` is TRUE (clause 12.2.7 derives FAULT from Reliability). The
   reading stops changing. Repair the wire and `Reliability` returns to
   `no-fault-detected`, FAULT clears, and the reading moves again.
4. **Failure Mode - a request it cannot serve.** Reading an object or property
   the sensor does not have, or an array element past the end, gets the error
   clause 15.5.1.3.1 names (`unknown-object`, `unknown-property`,
   `invalid-array-index`); a WriteProperty gets a **Reject**
   `unrecognized-service`, because a B-SS is read-only.

### The sensor Simulation

A real smart sensor samples a thermistor, a contact and a selector switch. This
example has none of them, so it **simulates** them - non-BACnet scaffolding,
outside the BACnet Surface, in section 2 of [`main.go`](main.go)
(`sensorSimulationTick()`, called beside `BACnetStack_Tick()`). Once a second:

- the **temperature** (Analog Input 1) moves 0.5 C, ramping up to 26.0 and back
  down to 18.0;
- the **contact** (Binary Input 1) toggles every 3 s;
- the **selector** (Multi-State Input 1) steps 1 -> 2 -> 3 -> 1 every 4 s.

The **broken wire** is a test-only hook, not a feature: if the environment
variable `CAS_EXAMPLE_TEST_SENSOR_FAULT_FILE` names a file, the wire counts as
broken for as long as that file exists. The Wire Tests create and delete it.
Nothing on the network can break the sensor.

To port the example to hardware, replace `sensorSimulationTick()` with your own
sampling (on a timer or another goroutine - never block the BACnet callbacks on
slow I/O), and set `Reliability` from your hardware's own diagnostics.

## What this example supports

The example implements exactly the capabilities below - and nothing more,
which is the point of a profile example.

### BIBBs (BACnet Interoperability Building Blocks)

| BIBB | Description | Supported |
|------|-------------|:---------:|
| DS-RP-B | Data Sharing - ReadProperty - B | ✅ |
| DM-DDB-B | Device Management - Dynamic Device Binding - B | ✅ |
| DM-DOB-B | Device Management - Dynamic Object Binding - B | ✅ |

### Services (executed / B-side)

| Service | Notes |
|---------|-------|
| ReadProperty | Responds to property reads (DS-RP-B). |
| Who-Is / I-Am | Answers Who-Is with I-Am, and broadcasts an I-Am on start-up (DM-DDB-B). |
| Who-Has / I-Have | Answers Who-Has with I-Have (DM-DOB-B). |

### Object types

| Object type | Instance | Name |
|-------------|:--------:|------|
| Device | 389001 | Chipkin Example B-SS |
| Analog Input | 1 | Bronze |
| Binary Input | 1 | Emerald |
| Multi-state Input | 1 | Hot Pink |
| Network Port | 1 | Vermilion |

Every required property of every object, and who answers it, is in
[docs/PICS.md](docs/PICS.md).

## Requires the CAS BACnet Stack (licensed product)

This example **builds against the CAS BACnet Stack, which is a commercial
Chipkin product** - it is not free or open source, and there is no
public/trial build. The stack is referenced here as the **private** git
submodule `submodules/cas-bacnet-stack` (tracking the `6.x` branch). The Go
edition loads the stack at run time as a **shared library**
(`CASBACnetStack_x64_Release.dll` on Windows / `libCASBACnetStack_x64_Release.so`
on Linux) that you build from that submodule - this repository and its releases
never contain it. **To get the CAS BACnet Stack (and access to build this
example), contact Chipkin:** <https://store.chipkin.com/services/stacks/bacnet-stack>
or sales@chipkin.com.

## What's in this repository

This is a **self-contained** project. It ships:

- `main.go` - the example device (section 2 is the sensor Simulation);
  `property_dispatch.go` - its section 3, the `Get*Property` callbacks.
- `cgobridge/` - the only package with `import "C"`: thin Go wrappers for the
  `BACnetStack_*` calls, the BACnet/IP transport callbacks, and the callback
  trampolines (see [How the Go binding works](#how-the-go-binding-works)).
- `common/` - the cgo-free Go helper (UDP socket, IPv4, command line, device
  state, BACnet constants).
- `tests/` - the Wire Tests ([tests/README.md](tests/README.md)), copied
  unchanged from the C++ edition; `wiretest/` - the series' shared Wiretest
  Harness 0.7.0, vendored verbatim.
- `docs/PICS.md` - the conformance statement; `docs/objects.json` - the input
  to its objects-and-properties generator.
- `submodules/cas-bacnet-stack/` - the **CAS BACnet Stack as a git submodule**
  (private; requires a license - see above). cgo includes its
  `source/CASBACnetStackDLL.h` in place; its MSVC / g++ build produces the
  shared library, in its own `bin/` folder.

## Prerequisites

- **Go 1.20 or later** (verified with go1.21.6), with **cgo**
  (`CGO_ENABLED=1`).
- **A C compiler for cgo.** On Windows that is **MinGW-w64 gcc** (for example
  from MSYS2, or the one bundled with Strawberry Perl - verified with
  `x86_64-w64-mingw32` gcc 13.2.0), separate from the MSVC toolchain that
  builds the stack's DLL. On Linux, gcc.
- **To build the stack's shared library:** on Windows, Visual Studio Build
  Tools (MSBuild, C++); on Linux, a C++17 g++.
- **Python 3** (standard library only) to run the Wire Tests.
- Git (to fetch the stack submodule).

## Build

```bash
git clone --recursive https://github.com/chipkin/BACnetProfileExample-B-SS-Go.git
cd BACnetProfileExample-B-SS-Go
```

Already cloned without `--recursive`? Run `git submodule update --init --recursive`.

### 1. Build the CAS BACnet Stack library from the submodule

The library goes into the submodule's own `bin/` folder, where the example
finds it at run time when it is run from this checkout. Do not commit it to this
repository (it is gitignored, and never part of a release).

**Windows** - the stack's MSVC project has a `ReleaseDll|x64` configuration
that builds exactly this DLL, under exactly this name (a customer-interface
build: no test-tool exports):

```powershell
& "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\MSBuild\Current\Bin\amd64\MSBuild.exe" `
  submodules\cas-bacnet-stack\projects\msvs\CASBACnetStack\CASBACnetStack.vcxproj `
  -p:Configuration=ReleaseDll -p:Platform=x64 -m
# -> submodules\cas-bacnet-stack\bin\CASBACnetStack_x64_Release.dll
```

(Adjust the MSBuild path to your Visual Studio / Build Tools install.)

**Linux** - build the shared library straight from the stack's sources:

```bash
mkdir -p submodules/cas-bacnet-stack/bin
g++ -shared -fPIC -std=c++17 -O2 \
    -I submodules/cas-bacnet-stack/source \
    submodules/cas-bacnet-stack/source/*.cpp \
    -o submodules/cas-bacnet-stack/bin/libCASBACnetStack_x64_Release.so
```

Only the **Release** library is used: the adapter loads `CASBACnetStack_x64_Release`
(`BACNET_LIB_FILE`), and there is no `DebugDll` configuration.

### 2. Build the example

```bash
CGO_ENABLED=1 go build -o build/BACnetExampleBSS .        # Linux
```

```powershell
$env:CGO_ENABLED = 1; go build -o build\BACnetExampleBSS.exe .   # Windows (gcc on PATH)
```

`build/BACnetExampleBSS` is the name the Wire Tests look for by default (the
C++ edition's name). `go vet . ./cgobridge ./common` should be clean; avoid
`./...`, which walks into the submodule's own Go files.

## Run

The executable loads the stack library at run time (not the operating system
at start-up), from the first of these folders that has it:

1. the **run directory** - the folder that holds the executable;
2. in a development checkout, `submodules/cas-bacnet-stack/bin/` (found by
   looking upwards from the executable's folder) - so `build/BACnetExampleBSS`
   in this checkout needs no set-up.

A **release** of this example ships the executable without the stack library
(it is a commercially licensed product). Run on its own, it stops with this
message on stderr and exit code 1:

```
ERROR: This example cannot run without the CAS BACnet Stack library.

  Missing file: CASBACnetStack_x64_Release.dll
  Looked in:    C:\BACnetExampleBSS

The CAS BACnet Stack is a commercially licensed (paid) library from Chipkin.
It is not included in this release, so the example cannot start without it.

  - To license the CAS BACnet Stack: https://store.chipkin.com/contact-us
  - Licensed customers: build the library from the cas-bacnet-stack submodule
    (see README.md) and place CASBACnetStack_x64_Release.dll next to this executable.
```

Licensed customers build the library (see [Build](#build)) and put
`CASBACnetStack_x64_Release.dll` (Windows) / `libCASBACnetStack_x64_Release.so`
(Linux) next to the executable. `--help` and `--version` work without it.

```bash
./build/BACnetExampleBSS --port 47821 --deviceID 12345
```

Expected output:

```
BACnet B-SS (Smart Sensor) Example - Go v1.1.0
CAS BACnet Stack version: 6.0.23.0
Common helper (common/) version: 1.1.0
FYI: Listening for BACnet/IP on UDP port 47821 (Network Port 1).
TX 21 bytes to <your-subnet-broadcast>:47821 (broadcast) (Network Port 1) - Unconfirmed: I-Am
FYI: Device 12345 ("Chipkin Example B-SS") ready. Vendor ID 389. Press 'h' then Enter for help.
```

The device listens on UDP **47808** (BACnet/IP) by default. Allow that port
through your firewall. To use a different port, pass `--port`.

### Command-line options

| Option | Default | Meaning |
|--------|---------|---------|
| `--port <n>` | `47808` | UDP port to listen on (BACnet/IP). |
| `--deviceID <n>` | `389001` | The device's BACnet instance number (BACnet requires this to be configurable). |
| `--help`, `-h` | - | Show usage and exit. |
| `--version` | - | Print the example, stack and `common/` versions, then exit. |

### Interactive commands

The console stays in its normal line mode, so press the key and then **Enter**
(the C++ edition reads single raw key presses - Go's standard library has no
portable way to; see [TUTORIAL.md](TUTORIAL.md#interactive-keys-are-a-simplification)):

| Key | Action |
|-----|--------|
| `h` | Show the version information and this command list. |
| `q` | Quit. |
| `u` | Nudge the simulated Analog Input 1 (`Bronze`) reading up by 1.1 (the C++ edition's up arrow). |
| `d` | Nudge it down by 1.1 (the C++ edition's down arrow). |

The Simulation carries on ramping from the nudged value, and a client
re-reading it sees the change. Ctrl+C also shuts the device down cleanly. With
stdin a pipe (as the Wire Tests run it), a `q` written to it quits at once.

## Verify

Use a BACnet client such as the
[**CAS BACnet Explorer**](https://store.chipkin.com/products/tools/cas-bacnet-explorer):

1. **Discover** - send a **Who-Is**. The device replies with **I-Am** from
   instance **389001** (vendor **389**). It also broadcasts an I-Am at
   start-up.
2. **Browse the object model** - the device shows five objects: the Device
   (`Chipkin Example B-SS`), the three sensors, and the Network Port
   (`Vermilion`). Reading the Device's `Object_List` returns all five.
3. **Read the Device** - ReadProperty `389001` -> `Object_Name` returns
   `"Chipkin Example B-SS"`; `Protocol_Revision` returns `30`; `Description`
   returns the profile description string.
4. **Read a sensor** - ReadProperty Analog Input `1` -> `Present_Value` returns
   the current simulated temperature (`21.5` at start-up, then 0.5 higher each
   second); read it again and it has moved. `Units` returns `degrees-Celsius`;
   `Reliability` returns `no-fault-detected`; `Status_Flags` returns all FALSE;
   `Out_Of_Service` returns `false`; `Object_Name` returns `"Bronze"`. Repeat
   for Binary Input `1` (`"Emerald"`, has `Polarity`) and Multi-State Input `1`
   (`"Hot Pink"`, has `Number_Of_States` = 3). Every required property of every
   object is readable.
5. **Confirm the profile boundary** - a **WriteProperty** to any object is
   rejected. That is correct: a B-SS Smart Sensor is read-only.

## Wire Tests

The checks above, automated: a Python BACnet client (the Peer) drives the
running example over BACnet/IP from outside it. Python 3 standard library only.
The scripts are the C++ edition's `tests/`, **unchanged**: the Go build is a
single executable at the path they expect (`build/BACnetExampleBSS`).

| Script | What it proves |
|---|---|
| `discovery_test.py` | Who-Is, the Device's `Object_List`, then for every object its `Property_List` and a ReadProperty of every property it lists - every read comes back as a value. |
| `sensing_test.py` | **The Profile Function.** Reads the three sensors again and again for 12 s; the readings follow the Simulation (0.5 C steps, a toggle every 3 s, states in order). Fails against a sensor that serves constants. |
| `failure_modes_test.py` | **The Failure Modes.** Breaks the simulated sensor wire: `Reliability` `open-loop`, `Status_Flags` FAULT, a frozen reading; repairs it: all clear, the reading moves. Then the errors for an unknown object / property / array index, and the Reject for a WriteProperty. |

Run them all after the build (the executable finds the library in the
submodule's `bin/`, see [Run](#run)):

```bash
python tests/run_wire_tests.py                    # default --exe build/BACnetExampleBSS(.exe), --port 47821
python tests/run_wire_tests.py --port 47821 --exe build/BACnetExampleBSS
```

It ends with `ALL WIRE TESTS PASSED` (exit 0); about 25 seconds.
[tests/README.md](tests/README.md) says what each check proves. **Verified on
Windows only** (3 runs in a row, 36/36 checks each); this port has not been
run on Linux. CI (`.github/workflows/release.yml`) runs them on Linux.

## How the Go binding works

If you have read the C++ edition, the object model and callback shapes are the
same; this section is only about what is mechanically different because Go
reaches the stack through cgo.

- **The stack's own adapter, in place, loaded at run time.**
  `cgobridge/stack_adapter.c` compiles the stack's generated C++ adapter
  (`submodules/cas-bacnet-stack/adapters/cpp/`) straight from the submodule in
  its DLL mode: every `BACnetStack_*` is a function pointer that
  `LoadBACnetFunctions()` binds with `LoadLibrary`/`GetProcAddress` (Windows)
  or `dlopen`/`dlsym` (Linux), after checking the stack's API version - the same
  call the C++ edition makes first thing in `main()`. The executable has no
  load-time dependency on the stack library (its import table does not list
  it), so a missing library is reported by the program itself (see
  [Run](#run)). Nothing from the stack is copied into this repository.
- **Callback trampolines.** cgo cannot hand a Go function to C, so every
  callback the stack calls goes C wrapper (`cgobridge/trampolines.c`, the
  header's exact signature) -> `//export`ed Go function
  (`cgobridge/callbacks.go`) -> the Go function `main.go` registered. The
  stack's own Go adapter (`adapters/golang/CASBACnetStackAdapter.go`) holds
  this same pattern, but at this Stack Pin it does not compile and is a
  `package main` sample, so this example carries its own - marked as a
  workaround, see [TODO.md](TODO.md).
- **Package-level state.** The callbacks are top-level functions reading
  `common.DeviceState` through a mutex (`common.WithState`) rather than
  variables local to `main()` - see `common/device_state.go`.
- **A standard-library transport.** `common/simple_udp.go` owns the UDP socket
  (`net.ListenUDP`); the stack pulls one datagram per receive callback, as in
  every edition.

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

- **ANSI/ASHRAE Standard 135** (BACnet) - the protocol standard. Object model
  (Clause 12), services (Clause 15), BACnet/IP (Annex J), device profiles
  (Annex L). Purchase / preview via the [ASHRAE store](https://www.ashrae.org/technical-resources/standards-and-guidelines).
- **What is BACnet?** - Chipkin's introduction:
  <https://docs.chipkin.com/protocols/bacnet/>.
- **CAS BACnet Stack** - product page and documentation:
  <https://store.chipkin.com/services/stacks/bacnet-stack>.
- **CAS BACnet Explorer** - client for testing this device:
  <https://store.chipkin.com/products/tools/cas-bacnet-explorer>.
- **Shared helper used by this example** - [`common/README.md`](common/README.md).

See also [TUTORIAL.md](TUTORIAL.md), [docs/PICS.md](docs/PICS.md),
[CHANGELOG.md](CHANGELOG.md), [TODO.md](TODO.md) and [AGENTS.md](AGENTS.md).
