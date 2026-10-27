# Changelog

All notable changes to this example. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

_No changes yet._

## [1.2.0] - 2026-10-27

Built on CAS BACnet Stack 6.x (`69b6249f`).

### Added

- **`--deviceName <text>`** sets the Device's Object_Name at start-up, like
  `--deviceID` sets its instance (ASHRAE 135-2024 cl. 12.1.1.3, 12.1.1.4).
  Object_Name must be unique on the internetwork, so two units could not share
  the compile-time name. The default is unchanged (`Chipkin Example B-SS`). A name
  must be 1..255 bytes with no control characters, otherwise the example warns and
  keeps the default. The name is not writable over the network. Documented in the
  README, PICS and TUTORIAL.
- **Wire Test** `tests/configuration_test.py` checks `--deviceID` and `--deviceName`
  over Who-Is, ReadProperty and Who-Has, and that `--help` lists exactly the
  options the example reads.

### Changed

- **Updated to CAS BACnet Stack 6.x `69b6249f`.** No change to the Go code.
- **Wire Tests default to this example's own port, 47822.** Runs that pass `--port`
  are unchanged.
- **PICS** now cites the CAS BACnet Stack's property profiles for the
  objects-and-properties tables.

### Fixed

- **PICS: Database_Revision is 5**, not 0. The stack adds one per object created at
  start-up (cl. 12.11): the Device and its four objects.

### Compatibility

- **The stack library is built with the stack's CMake presets.** The stack no
  longer ships Visual Studio or Linux makefile projects. On Windows use
  `cmake --preset windows-x64-dll`, `cmake --build --preset windows-x64-dll-release`
  and `cmake --install build-windows-x64-dll --config Release` (the `-vs2022`
  presets for Visual Studio 2022); on Linux use `linux-x64-release-shared`. The
  install step puts the library in the submodule's `bin/`, where the example
  looks for it. CMake is now a prerequisite. See the README.
- **Go still needs cgo and a C compiler** (MinGW-w64 gcc on Windows, gcc on
  Linux), separate from the compiler that builds the stack library.

## [1.1.0] - 2026-10-16

Built on CAS BACnet Stack 6.x (`64b2cfac`, 6.0.23). This is the first tagged
release; it includes everything from the earlier 1.0.x development builds.

The sensor now performs the B-SS Profile Function with live readings and a
failure mode, matching the C++ example 1.3.0.

### Added

- **First Go B-SS (Smart Sensor) example** for the CAS BACnet Stack. It has a
  Device (instance 389001), Analog Input 1, Binary Input 1, Multi-State Input 1
  and Network Port 1; ReadProperty, Who-Is / I-Am and Who-Has / I-Have; options
  `--port`, `--deviceID`, `--version` and `--help`; interactive commands; and a
  clean Ctrl+C shutdown. See `docs/objects.json` and `docs/PICS.md`.
- **Sensor simulation.** Once a second Analog Input 1 ramps between 18.0 and 26.0
  C in 0.5 C steps, Binary Input 1 toggles every 3 s and Multi-State Input 1 steps
  1, 2, 3 every 4 s.
- **Analog Input 1 `Reliability`** reads `open-loop` while a simulated sensor wire
  is broken, and the FAULT bit of `Status_Flags` follows (cl. 12.2.7, 12.2.9).
- **Wire Tests** under `tests/` (discovery, sensing, failure modes), run against
  the built example (`build/BACnetExampleBSS`).
- `TODO.md` records a defect in the stack's Go adapter that this example works
  around.

### Changed

- **Protocol_Revision 30** and **segmentation supported**, from the stack.
- **The Device is named "Chipkin Example B-SS"** (was "Rainbow") so examples on
  the same network can be told apart.
- **The stack's adapter is used in place.** The stack's generated C++ adapter
  (`adapters/cpp/`) is compiled through cgo from the submodule; the stack's own Go
  adapter does not compile at this pin, so `cgobridge/` supplies the callback
  trampolines. No copy of the adapter is vendored.
- **The stack library is loaded at run time and releases ship without it.** It is
  looked up next to the executable, then in `submodules/cas-bacnet-stack/bin/`
  in a development checkout. If it is missing the example prints a clear message
  (the file name and every folder searched, that the stack is a licensed Chipkin
  product) and exits with code 1. `--help` and `--version` work without it.
- Start-up follows the C++ example: `--help` / `--version` before the socket, the
  version banner, the listening port, the start-up I-Am, then `ready`. Interactive
  commands (`h`, `q`, `u`, `d`, then Enter) are read from stdin, and a `q` written
  to a pipe quits at once.
- `common/` 1.1.0 (see `common/CHANGELOG.md`).

### Fixed

- **Network Port 1 properties a client could not read** (`IP_DNS_Server`,
  `FD_BBMD_Address`, `FD_Subscription_Lifetime`) now return values. Fixed by the
  stack update.
- **`Application_Software_Version` and `Firmware_Revision` were stale.** The first
  now follows the example's version, the second is built from the running stack's
  version.

### Compatibility

- Build with explicit package paths (`go build .`, `go vet . ./cgobridge ./common`), not
  `./...`, which recurses into the stack submodule and fails.
