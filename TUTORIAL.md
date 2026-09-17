# TUTORIAL - extending this example

This walks through the two most common changes: adding a second Analog
Input, and understanding the interactive-command simplification.

## Add a second analog input

Say you want Analog Input 2, "Sapphire", degrees Fahrenheit, starting at
72.0.

1. **`common/constants.go`** - add its instance number and name near
   `AnalogInputInstance`:

   ```go
   AnalogInput2Instance uint32 = 2 // "Sapphire"
   ```

2. **`common/device_state.go`** - add its live value to `DeviceState` (it's
   mutable, like `AnalogInput1Value`):

   ```go
   AnalogInput2Value float32
   ```

   and its start value in the `state` initializer:

   ```go
   AnalogInput2Value: 72.0,
   ```

3. **`main.go`** - add it in the "Add the read-only sensor objects"
   section:

   ```go
   if !bacnet.AddObject(deviceIDValue, common.ObjectTypeAnalogInput, common.AnalogInput2Instance) {
       return fail("Failed to add Analog Input %d (Sapphire).", common.AnalogInput2Instance)
   }
   ```

4. **`cas_bacnet_stack/property_dispatch.go`** - route its three served
   properties:

   - `getPropertyReal`: add an `objectInstance == common.AnalogInput2Instance`
     branch reading `s.AnalogInput2Value`.
   - `getPropertyEnumerated`: add a branch for its `Units` (Fahrenheit -
     look up the value in
     `submodules/cas-bacnet-stack/source/BACnetEngineeringUnits.h`, add a
     `common.EngineeringUnitsDegreesFahrenheit` constant for it).
   - `getPropertyCharString`: add `"Sapphire"` to the `Object_Name` switch.

**THIS DOES NOT FAIL LOUDLY IF YOU MISS A STEP.** `BACnetStack_AddObject`
succeeding does not mean every property is correctly served - a missed
`Get*Property` branch means the stack falls back to a stack-generated
default (often `0` or an empty string) with **no error on the wire and no
log line**. The only way to catch a half-added object is to read it back
with a real BACnet explorer (or re-check `docs/objects.json`/`docs/PICS.md`
by hand) after every object you add - see AGENTS.md "How to verify a
change".

If you also want it discoverable in `docs/PICS.md`/`docs/objects.json`,
regenerate or hand-edit those the same way the C++ edition's
`tools/gen-objects-properties.py` does (see that script's own docs) - this
repository does not (yet) have a Go-side generator; PICS.md's `<!--
OBJECTS-PROPERTIES:BEGIN/END -->` markers show exactly which block to
regenerate.

## Interactive commands are a simplification

`main.go`'s `spawnCommandReader()` reads **line-buffered** commands (type
`up`, press Enter) from a background goroutine over a channel, rather than
polling raw single keypresses the way the C++ edition does. This mirrors
the Rust edition's own choice, for the same underlying reason: Go's
standard library, like Rust's, has no portable non-blocking
single-keypress read without a third-party package
(`golang.org/x/term` would work, but this example deliberately stays
dependency-free - `go.mod` has no `require` lines at all). If you want raw
keypresses, add `golang.org/x/term` and replace `spawnCommandReader()`'s
`bufio.Scanner` loop with a raw-mode reader; keep the same channel-based
hand-off to the main tick loop so nothing but that one goroutine ever
touches the terminal, and nothing but the main loop ever touches the
stack (see AGENTS.md "Conventions" - the stack is single-threaded by
contract).

## Adding a new object type entirely

If you add an object type this example doesn't have yet (e.g. an Analog
Value), check `submodules/cas-bacnet-stack/source/CASBACnetStackDLL.h`
for whether it needs its own `BACnetStack_Add*Object` call (some object
types, like Trend Log, do) versus the generic
`BACnetStack_AddObject(deviceInstance, objectType, objectInstance)` every
object in this example currently uses. Add its `OBJECT_TYPE_*` constant to
`common/constants.go`, and route every property it requires in
`cas_bacnet_stack/property_dispatch.go` - re-check
`submodules/cas-bacnet-stack/docs/property-profile-reference.md` (in the
submodule) for the exact REQUIRED property list per clause 12, the same
source `docs/objects.json`/`docs/PICS.md` were built from.
