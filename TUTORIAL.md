# Tutorial - extending and reviewing the B-SS example

[README.md](README.md) says what this example *is*. This document is the *how*:
how to extend it into your own device, who serves which property, how to review
the result for conformance, and what goes wrong when you get it subtly right.

Read this once before you start changing `main.go` / `property_dispatch.go`. The
most expensive mistake in this example is silent, and the section it lives in is
[Add a second analog input](#add-a-second-analog-input).

- [Extending the example](#extending-the-example)
- [What each object type needs you to serve](#what-each-object-type-needs-you-to-serve)
- [Who serves what: the application or the stack?](#who-serves-what-the-application-or-the-stack)
- [Interactive keys are a simplification](#interactive-keys-are-a-simplification)
- [Reviewing your device](#reviewing-your-device)
- [Troubleshooting](#troubleshooting)

## Extending the example

The example is intentionally small so it's easy to change. Where things are:

| File | Holds |
|---|---|
| `main.go` | section 1 configuration, section 2 the sensor Simulation, the interactive commands, section 4 `run()` (device and object set-up, the tick loop) |
| `property_dispatch.go` | section 3: the `getProperty*` callbacks - which object/property is served with what |
| `common/constants.go` | BACnet enumeration values and the device identity (name, vendor, model, instances) |
| `common/device_state.go` | the live values the callbacks hand back (`common.DeviceState`) |
| `cgobridge/` | the only cgo code: `BACnetStack_*` wrappers, transport, callback trampolines |

**Change a sensor's value or name** - edit the constants / callbacks (e.g. the
Simulation's range and rates in `main.go` section 2, or the `"Bronze"` string in
`getPropertyCharString`).

**Replace the Simulation with your hardware** - `sensorSimulationTick()` in
`main.go` section 2 is the only code that changes the sensor values; the
`getProperty*` callbacks just hand back `AnalogInput1Value`, `BinaryInput1Value`
and `MultiStateInput1Value` from `common.DeviceState`. Replace the function body
with your own sampling and keep the callbacks as they are. Sample on a timer or
another goroutine and only *store* the result (through `common.WithState`): the
callbacks run inside `BACnetStack_Tick()`, so a callback that waits for an ADC
conversion or an I2C transaction delays every BACnet request. Delete
`sensorFaultFileVariable` and its environment-variable code - it is a test hook,
not something to ship.

**Report a sensor fault** - `Reliability` is how a BACnet sensor says "do not
trust this reading" (cl. 12.2.9). It is optional, so it takes two steps: enable
it with `cgobridge.SetPropertyEnabled(... common.PropertyIdentifierReliability ...)`
and serve it from `getPropertyEnumerated`. The stack then builds the FAULT bit of
`Status_Flags` from it by itself (cl. 12.2.7) - you never serve `Status_Flags`.
Once it is enabled, **always** answer it: the stack reads `Reliability` through
your callback every time a client reads `Status_Flags`, and a callback that
declines does not fail loudly - the stack substitutes the datatype default, `0`,
which is `no-fault-detected`. A forgotten branch therefore reports a healthy
sensor forever, with FAULT never set. (`failure_modes_test.py` is what catches
that: it breaks the sensor and expects the fault.) Set it from your hardware's
own diagnostics - `open-loop` (4) for a broken wire, `shorted-loop` (5),
`over-range` (2) / `under-range` (3) for a reading on the rails, `no-sensor` (1)
for a sensor that stopped answering.

**Change the device identity before you ship** - vendor ID, vendor name, model
name, description and device name are in `common/constants.go` (the "CHANGE ALL
OF THIS BEFORE YOU SHIP" block); `Application_Software_Version` is `appVersion`
in `main.go` and `Firmware_Revision` is the stack's own version. `Object_Name`
must be unique across the BACnet internetwork, and here it is a compile-time
constant while the instance is runtime-configurable (`--deviceID`) - make it
per-unit configurable in a real product.

### Add a second analog input

Read this whole recipe before starting - the last step is the one that is easy
to miss and the one BTL will fail you for.

> **Why there are four edits, not three - and why skipping one is SILENT.**
> Most of the `getProperty*` callbacks match on **both** object type *and*
> instance (`objectInstance == common.AnalogInputInstance`), so a new instance
> falls through every one of them. `getPropertyBool` is the exception: it
> matches on type only, so `Out_Of_Service` works for a new instance for free.
>
> Falling through a callback does **not** reliably produce an error. The stack
> errors only for the few properties it refuses to invent - `Present_Value`,
> `Number_Of_States`, `Relinquish_Default`, `Local_Date`, `Local_Time`, and a
> Network Port's `APDU_Length`. For everything else it **silently substitutes a
> default**:
>
> | Property | If you forget to serve it | Loud? |
> |---|---|:--:|
> | `Present_Value` | Error (`read-access-denied`) | yes |
> | `Object_Name` | reads back as the string **`"undefined"`** | **no** |
> | `Units` | reads back as **`no-units` (95)** | **no** |
> | `Out_Of_Service` | served on type alone - works by accident | n/a |
>
> **Doesn't the `errorCode` out-parameter fix this?** Only where it is right to.
> The stack presets it to `success` and reads it only when a callback declines,
> so you *can* turn any decline into a chosen BACnet error - but ending every
> callback with `*errorCode = unknown-property` breaks the device: the stack's
> decline-and-fabricate path is what answers required properties an application
> is not expected to serve (the Device's `Max_APDU_Length_Accepted`,
> `APDU_Timeout`, `Number_Of_APDU_Retries`). `property_dispatch.go` sets it in
> exactly one place, `State_Text` with an out-of-range array index.
>
> The object's `Property_List` **still advertises `Units` (117)**, so a
> half-added object does not look broken; it looks **healthy**. Add two of them
> and both report `Object_Name "undefined"` - duplicate object names inside one
> device, a spec violation and a hard BTL failure. **"It scanned OK" is exactly
> the failure mode, not evidence against it.**

```go
// 1) common/constants.go - a new instance number.
//    Naming: a second object of a type is "<Colour> 2" - so Analog Input 2 is
//    "Bronze 2", NOT a new colour. Each object TYPE owns one colour series-wide.
AnalogInput2Instance uint32 = 2 // "Bronze 2"

// common/device_state.go - its live value (and a start value in `state`).
AnalogInput2Value float32

// 2) main.go run() - add the object, next to the other AddObject calls.
//    Check the return, like every other stack call.
if !cgobridge.AddObject(deviceInstance, common.ObjectTypeAnalogInput, common.AnalogInput2Instance) {
	return fail("Failed to add Analog Input %d (Bronze 2).", common.AnalogInput2Instance)
}

// 3) property_dispatch.go - serve its Present_Value + Object_Name:
//    getPropertyReal:        AI/2 + Present_Value -> s.AnalogInput2Value
//    getPropertyCharString:  AI/2 + Object_Name   -> "Bronze 2"

// 4) DO NOT SKIP: serve its Units, in getPropertyEnumerated.
//    Units is a REQUIRED property of an Analog Input. The existing check reads
//    objectInstance == common.AnalogInputInstance, which is instance 1 - so
//    without this, Analog Input 2's Units silently reads no-units. It still
//    appears in the Object_List and its Present_Value reads back perfectly.
//    getPropertyEnumerated:  AI/2 + Units -> common.EngineeringUnitsDegreesCelsius
```

Then re-run the README's Verify steps **against Analog Input 2**, not just Analog
Input 1 - read every required property and **diff it against Analog Input 1**.
Any property that comes back `"undefined"`, `no-units`, or `0` where object 1
returns something real is a step you missed. Update `docs/objects.json` in the
same change (see [Keeping the PICS honest](#keeping-the-pics-honest)).

## What each object type needs you to serve

The application must serve every REQUIRED property the stack does not generate.
It differs per type - this is the checklist, so you do not have to infer it:

| Object type | You must serve | Plus |
|---|---|---|
| Analog Input | `Present_Value` (Real), `Object_Name`, `Units` | - |
| Binary Input | `Present_Value` (Enumerated), `Object_Name` | `Polarity` |
| Multi-State Input | `Present_Value` (Unsigned), `Object_Name` | `Number_Of_States` |

An object type this example does not have yet: check
`submodules/cas-bacnet-stack/source/CASBACnetStackDLL.h` for whether it needs
its own `BACnetStack_Add*Object` call (add a wrapper for it in
`cgobridge/stack.go`) and the stack's `docs/property-profile-reference.md` for
its required properties.

## Who serves what: the application or the stack?

For Analog Input 1, the whole picture:

| Property | Served by | How |
|---|---|---|
| `Object_Identifier` | **stack** | generated from the object you added |
| `Object_Type` | **stack** | generated |
| `Object_List` | **stack** | generated (Device object) |
| `Property_List` | **stack** | generated |
| `Status_Flags` | **stack** | generated - FAULT from `Reliability`, the other three bits FALSE here |
| `Reliability` | **you** | `getPropertyEnumerated` (optional; enabled in `run()`) - the Simulation's broken wire sets it to `open-loop` |
| `Event_State` | **stack**, sort of | no intrinsic alarming here, so nothing serves it - it reads `normal` only because `normal` is the enumeration's zero value. Correct by coincidence, not design. |
| `Out_Of_Service` | **you** | `getPropertyBool` - matched on object **type only** |
| `Present_Value` | **you** | `getPropertyReal` |
| `Object_Name` | **you** | `getPropertyCharString` |
| `Units` | **you** | `getPropertyEnumerated` |

Every object, not just this one, is in [docs/PICS.md](docs/PICS.md).

Going beyond reading (writable points, outputs, COV, alarms) means implementing a
richer profile - see the series table in [README.md](README.md).

## Interactive keys are a simplification

The C++ edition reads single raw key presses (`_kbhit()`/`_getch()` on Windows, a
raw-mode terminal on POSIX), including the arrow keys. Go's standard library has
no portable raw-keyboard read, and this example stays dependency-free (`go.mod`
has no `require` lines). So `spawnCommandReader()` in `main.go` reads stdin byte
by byte while the terminal stays in its normal line mode: press the key, then
Enter. `u` and `d` stand in for the up and down arrows. A `q` written to a pipe
(the Wire Tests stop the example that way) quits at once.

The reader is one goroutine that blocks on stdin and hands each command to the
main loop over a channel; only the main loop ever calls the stack. If you want
raw key presses, add `golang.org/x/term`, put the terminal in raw mode inside
that goroutine, and keep the channel hand-off.

## Reviewing your device

After you have changed anything, review it against the conformance statement
rather than against "it looked fine in the explorer":

1. Regenerate [docs/PICS.md](docs/PICS.md) after editing `docs/objects.json`
   (see below). A ⚠ row is a required property nothing serves.
2. Read **every** property listed for **every** object with a BACnet client, and
   compare the value against the PICS. `tests/run_wire_tests.py` reads them all
   and fails on any that does not return a value (README "Wire Tests") - keep it
   passing, then check the values yourself. `"undefined"`, `no-units` and `0`
   are the three shapes a missed callback takes. Its other two scripts check
   that the readings change and that a fault reads as `Reliability` + FAULT - if
   you replace the Simulation with hardware, they show you whether your
   sampling reaches the wire.
3. Diff a new object of a type against the existing one of that type. Anything
   that differs and shouldn't is a callback that matched on instance.
4. Confirm the services you do **not** implement are still rejected - for B-SS,
   WriteProperty to any object.

### Keeping the PICS honest

`docs/PICS.md` is partly generated. `docs/objects.json` describes each object and
who serves which property; the series tool regenerates the object tables from it
plus the stack's own `docs/property-profile-reference.md` at the pinned commit:

```bash
python tools/gen-objects-properties.py BACnetProfileExample-B-SS-Go            # rewrite
python tools/gen-objects-properties.py BACnetProfileExample-B-SS-Go --check    # fail if stale
```

(That tool lives in the example-series repository, not in this one. If you only
have this repository, edit the generated block by hand and keep it matching the
callbacks in `property_dispatch.go`.)

## Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| `ERROR: This example cannot run without the CAS BACnet Stack library.` (exit code 1) | The stack library is in none of the folders the message lists. Build it from the submodule (README "Build", step 1) and put it next to the executable, or run the executable from this checkout (it also looks in `submodules/cas-bacnet-stack/bin/`). |
| `Error: failed to load the CAS BACnet Stack from <folder>: ...` | The library was found but could not be used: a different architecture (it must be x64), a stack from another major version (`stack API major version ...`), or a missing export (`missing export: ...` - a wrong or corrupt build). Rebuild it from the submodule at this Stack Pin. |
| `go build`: `cgo: C compiler "gcc" not found` | cgo needs a gcc. On Windows install MinGW-w64 (MSYS2) and put its `bin` on `PATH`. |
| `go build`: `CASBACnetStackDLL.h: No such file or directory` | Submodules not checked out: `git submodule update --init --recursive`. |
| `go build ./...` or `go vet ./...` fails inside `submodules/` | `./...` walks into the stack's own Go files. Name the packages: `go vet . ./cgobridge ./common`. |
| App prints *"Failed to bind UDP port 47808"* | Another BACnet program is already using 47808. Stop it, or run with `--port <n>`. |
| Client sends Who-Is but sees no I-Am | Firewall is blocking UDP 47808, or the client and device are on different subnets (Who-Is is a broadcast). Allow the port; test on the same subnet first. |
| Typing `q` does nothing | Press Enter after the key - the console is in line mode (see above). |
