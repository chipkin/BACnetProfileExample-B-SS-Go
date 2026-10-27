# AGENTS.md

Guidance for AI coding agents working in this repository. See
<https://agents.md/> for the format. Human contributors should read
[README.md](README.md) first, then [TUTORIAL.md](TUTORIAL.md).

## What this project is

A **tutorial** Go example that implements the BACnet **B-SS (Smart Sensor)**
device profile using the CAS BACnet Stack through cgo. It is the Go port of
`BACnetProfileExample-B-SS-CPP` (v1.3.0) - same Device, objects, properties,
Simulation, Failure Modes, test-only hook and command line - and the C++
edition's Wire Tests run against it unchanged. The code must read like a
tutorial a customer can learn from and copy-paste. Favour clarity over
cleverness, and keep it a faithful translation of the C++ `main.cpp`.

## Layout

- `main.go` - sections 1, 2 and 4 of the C++ `main.cpp`: configuration, the
  sensor **Simulation** (`sensorSimulationTick()`), the interactive commands,
  and `run()` (device and object set-up, the tick loop).
- `property_dispatch.go` - section 3: the `getProperty*` callbacks.
- `cgobridge/` - the **only** package with `import "C"`. `stack.go` holds the
  cgo flags, the library search and load, and the `BACnetStack_*` wrappers;
  `stack_adapter.c` compiles the stack's generated `adapters/cpp/` adapter
  **in place** from the submodule (DLL mode, run-time loading);
  `callbacks.go` the `//export`ed callbacks and the BACnet/IP transport;
  `trampolines.c` the C halves of the callbacks.
- `common/` - cgo-free Go helper (UDP socket, IPv4, command line, BACnet
  constants, device state), with its own version (`CommonVersion`) and
  `common/CHANGELOG.md`. Never add `import "C"` here.
- `tests/` - the Wire Tests, the C++ edition's `tests/` with this port's
  default port (47822) and `main.go` named where they name `main.cpp`.
- `wiretest/` - the series' Wiretest Harness 0.7.0, byte-identical to
  `BACnetProfileExample-B-SS-CPP/wiretest/`. **Never edit it**; report a
  needed change instead.
- `docs/PICS.md` / `docs/objects.json` - the conformance statement; its
  objects-and-properties block is generated from `docs/objects.json` by the
  series tool. Update `objects.json` with any callback change.
- `submodules/cas-bacnet-stack/` - the CAS BACnet Stack (private submodule,
  Stack Pin `6.x` @ `69b6249f`). **Never edit anything under it.**
- `TODO.md` - known stack/adapter defects this example works around.

## The adapter rule

The stack's language adapters are used **in place** from the submodule, never
copied, regenerated or hand-written into this repository. For Go that means:
cgo `#include`s `CASBACnetStackDLL.h` from the submodule and generates the
bindings itself. The stack's `adapters/golang/` is an importable package at the pin, but
this port has not switched to it (see TODO.md), so `cgobridge/callbacks.go` and
`cgobridge/trampolines.c` carry the callback trampolines it provides, marked as a
workaround. Do not patch the submodule; when this port switches to the adapter,
replace the workaround with it.

## Build

```bash
git submodule update --init --recursive
cd submodules/cas-bacnet-stack
# Windows: cmake --preset windows-x64-dll && cmake --build --preset windows-x64-dll-release && cmake --install build-windows-x64-dll --config Release
# Linux:   cmake --preset linux-x64-release-shared && cmake --build --preset linux-x64-release-shared && cmake --install build-linux-x64-release-shared
cd ../..
CGO_ENABLED=1 go build -o build/BACnetExampleBSS .     # .exe on Windows
go vet . ./cgobridge ./common
```

`go build` does not build the stack's shared library - that is the separate
CMake step in README "Build". CMake is the stack's only build: it no longer has
`projects/msvs/` or `projects/linux/` (removed before `6.x` @ `29b83de6`).
`cmake --install` puts the library in the submodule's `bin/`, where the example
loads it from. Always a customer-interface build, never the test-tool library.
A gcc must be on `PATH` for cgo (MinGW-w64 on Windows). Do not use `./...`: it
walks into the submodule's own Go files. **Never commit or release the stack's
DLL / `.so`** - it lives in the submodule's `bin/` (gitignored there), and
`git ls-files | grep -iE '\.(dll|so|node|lib|a)$'` must print nothing.

## Run

```bash
./build/BACnetExampleBSS [--port 47808] [--deviceID 389001] [--deviceName "Chipkin Example B-SS"]
```

The stack library is loaded at run time (the stack's `adapters/cpp/` in DLL mode,
compiled in place by `cgobridge/stack_adapter.c`) from the executable's folder,
else from `submodules/cas-bacnet-stack/bin/`; without it the program prints the
series' missing-library message (rule 2b) and exits 1. Interactive commands: `h`, `q`, `u`, `d`, each followed by
Enter.

## Conventions

- Device "Chipkin Example B-SS", instance 389001, vendor 389; objects use the
  series' colour names. Protocol_Revision 30.
- Implement **only** what the B-SS profile requires, but serve **every required
  property** of each object.
- The Simulation is outside the BACnet Surface. Its broken-wire hook is
  test-only (`CAS_EXAMPLE_TEST_SENSOR_FAULT_FILE`, an environment variable) -
  never a command-line option or a menu key.
- Every stack call's return value is checked.
- Callbacks are top-level functions reading `common.DeviceState` through
  `common.WithState`; only the main loop calls the stack.
- Callback C definitions live in `cgobridge/trampolines.c`, not in a cgo
  preamble (a file with `//export` functions has its preamble compiled twice).

## How to verify a change

Run the Wire Tests after any change (README "Wire Tests"):

```bash
python tests/run_wire_tests.py --port <free port>
```

A new check must fail with its feature removed (mutate, rebuild, confirm the
executable changed, see the check fail, restore). If you changed the objects or
their properties, update `docs/objects.json` and regenerate `docs/PICS.md` with
the series tool (`python tools/gen-objects-properties.py BACnetProfileExample-B-SS-Go`
from the series root).

## Releasing

Bump `appVersion` in `main.go` and add an entry to [CHANGELOG.md](CHANGELOG.md),
then tag `vX.Y.Z`. `.github/workflows/release.yml` builds the stack library (to
compile and test only), the example, runs the smoke test and (Linux) the Wire
Tests, and publishes the example's executables, metrics and docs - never the
stack library.

## License

This example's own code is CC0-1.0 - see [LICENSE](LICENSE). The CAS BACnet
Stack (`submodules/cas-bacnet-stack`) is a separate, commercially licensed
product and is not covered by it.
