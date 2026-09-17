# `cas_bacnet_stack/` - vendored Go adapter

The CAS BACnet Stack's Go cgo adapter, copied (not referenced) from
`submodules/cas-bacnet-stack/adapters/golang/CASBACnetStackAdapter.go`, then
genuinely **fixed** to compile against this repository's pinned `6.x`
header (`source/CASBACnetStackDLL.h`) and extended to cover this example's
device model. Never edit the submodule's copy - only this vendored one.

## What's here

| File | What it is |
|---|---|
| `cas_bacnet_stack_adapter.go` | The vendored skeleton, fixed (see below) and extended: cgo preamble (declarations only), callback registration, the `//export`ed transport/time/`Get*Property` callback shims, and thin Go wrapper functions (`AddDevice`, `AddObject`, `Tick`, `SendIAm`, ...) that let `main.go` drive the stack without its own `import "C"`. |
| `cas_bacnet_stack_trampolines.c` | The C-side halves of each callback trampoline (see its own header comment for **why** this is a real, separately-compiled `.c` file rather than function bodies inside the cgo preamble comment - a toolchain finding, not a style choice). |
| `property_dispatch.go` | The property-to-callback dispatch logic for this example's specific device model (which object/property maps to which value) - mirrors the Rust sibling's `main.rs` `get_property_*` functions field-for-field. |

Not vendored: `submodules/cas-bacnet-stack/adapters/golang/propertybufferhelper/`
- like the C#/Python/Rust editions' equivalent unused pack-helper files, it
packs buffer-shaped `Send*` calls (ReadProperty/WriteProperty/CreateObject
**as a client**) that this read-only B-SS example never makes.

## The skeleton shipped broken - what was fixed

The task that produced this repository described these as **known-broken,
must-fix** items in the vendored skeleton, verified against
`submodules/cas-bacnet-stack/source/CASBACnetStackDLL.h` at the pinned
commit (`ec60c71801aad076a46c0ec783c5566a6cd0f3a0`, tag
`v5.3.4-711-gec60c718`):

### Fix #1 - `RegisterCallbackReceiveMessage`/`SendMessage` renamed to `*ForPort`

The skeleton called `BACnetStack_RegisterCallbackReceiveMessage` /
`BACnetStack_RegisterCallbackSendMessage`. **Neither symbol exists in the
current header.** They were renamed/widened to:

```c
DllExport void BACnetStack_RegisterCallbackReceiveMessageForPort(uint16_t(*FPCallbackReceiveMessageForPort)(uint8_t* message, const uint16_t maxMessageLength, uint8_t* sourceConnectionString, uint8_t* sourceConnectionStringLength, uint8_t* destinationConnectionString, uint8_t* destinationConnectionStringLength, const uint8_t maxConnectionStringLength, uint32_t* networkPortInstance));
DllExport void BACnetStack_RegisterCallbackSendMessageForPort(uint16_t(*FPCallbackSendMessageForPort)(const uint8_t* message, const uint16_t messageLength, const uint8_t* connectionString, const uint8_t connectionStringLength, const uint32_t networkPortInstance, bool broadcast));
```

- the trailing parameter widened from `uint8_t networkType` to
  `uint32_t* networkPortInstance` (receive, an OUT param) / `uint32_t
  networkPortInstance` (send, not a pointer).

Both trampolines were rewritten to the new names and signatures (all three
layers: the `extern` C forward declarations in the cgo preamble, the C
wrapper in `cas_bacnet_stack_trampolines.c`, and the Go `//export`ed
function). Because this is a **single-transport (BACnet/IP only)** B-SS
device, `goCallbackReceiveMessageForPort`'s Go-side implementation writes
`common.NetworkPortInstance` (this example's one Network Port, "Vermilion")
into the `networkPortInstance` out-param on every call, and
`goCallbackSendMessageForPort` ignores the incoming `networkPortInstance`
parameter entirely - there is nothing to branch on with only one port. This
matches the single-transport convention the C#/Python/Rust editions of this
series document for their own equivalent callbacks.

### Fix #2 - registrations no longer type-erase with `(*[0]byte)`

The skeleton's registrations mostly cast the callback pointer through
`(*[0]byte)(C.fpCallbackXxx)`, bypassing any compile-time signature check
against the real header - only `ReceiveMessage` had already been "fixed" to
pass the trampoline directly, per a prior bug-fix comment in the file.

**As specified, this fix asks for every registration to pass the C
trampoline directly, with no type-erasing cast, so cgo type-checks each one
against the real header.** That does not compile as literally stated on
the Go toolchain actually available in this repository
(`go1.26.3` - see `RegisterCallbacks()`'s own doc comment in
`cas_bacnet_stack_adapter.go` for the exact mechanism): a bare reference to
a C function used as a *value* (`C.fpCallbackXxx`, not called) is
represented by this cgo version as an `unsafe.Pointer`-producing
expression, and Go's assignability rules do not implicitly convert
`unsafe.Pointer` to the named `*[0]byte` parameter type
`BACnetStack_RegisterCallback*` expects - so an explicit
`(*[0]byte)(unsafe.Pointer(...))` conversion is required on **every**
registration to link at all, not just the ones the skeleton already cast.
This was confirmed empirically while building this example (not assumed -
see `CHANGELOG.md`/`AGENTS.md` "verified findings"), and is a toolchain
representation change, not a design regression.

The real type-safety win survives regardless: every `fpCallbackXxx` C
trampoline in `cas_bacnet_stack_trampolines.c` is declared with the
**exact** parameter types copied from the current header, so a future
header change to any callback signature fails this file's own C
compilation - long before it could reach a silent runtime ABI mismatch.
Applied consistently to every callback this example registers, old and new
alike (not just the ones this fix touched).

### Fix #3 - `goCallbackGetSystemTime`'s extern declaration used `time_t`, not `CASBACnetTime`

The skeleton declared `extern time_t goCallbackGetSystemTime();`. The
header's `RegisterCallbackGetSystemTime` takes:

```c
typedef int64_t CASBACnetTime;
DllExport void BACnetStack_RegisterCallbackGetSystemTime(CASBACnetTime(*FPCallbackGetSystemTime)());
```

`time_t` is platform-defined (and on 32-bit Windows historically 32-bit) -
not guaranteed to match `int64_t`. Every layer (`extern` declaration, C
wrapper, Go `//export` function) now uses `CASBACnetTime` throughout, and
the Go side returns Unix epoch **seconds** as an `int64`.

### Fix #4 - only 2 of 6 needed `Get*Property` callbacks were wired

The skeleton only registered `GetPropertyCharacterString` and
`GetPropertyReal`. This example's device model (see the top-level
`docs/objects.json`) also needs:

| Callback | Used for |
|---|---|
| `GetPropertyBool` | `Out_Of_Service` on every input object and the Network Port |
| `GetPropertyEnumerated` | Binary Input `Present_Value`/`Polarity`, Analog Input `Units`, Network Port `BACnet_IP_Mode` |
| `GetPropertyUnsignedInteger` | Multi-State Input `Present_Value`/`Number_Of_States`, Device `Vendor_Identifier`, Network Port `Max_APDU_Length_Accepted`/`Reference_Port`/`BACnet_IP_UDP_Port` |
| `GetPropertyOctetString` | registered for completeness/parity with the header's full `Get*Property` set (this device model does not currently need it - the Network Port's IPv4 addressing is served via `GetPropertyOctetString`'s C counterpart in the C++ edition, but this stack pin answers `IP_Address`/`IP_Subnet_Mask` through the same octet-string callback in the Rust sibling; this edition's `goCallbackGetPropertyOctetString` is wired and registered but always declines - see its own comment for exactly why and what to change if you add an object that needs it) |

All six are now registered in `RegisterCallbacks()`, and their dispatch
logic lives in `property_dispatch.go`, mirroring the Rust sibling's
`main.rs` `get_property_*` functions' property-to-callback mapping for the
same B-SS model.

### Fix #5 - `LDFLAGS` pointed at a Debug library that doesn't ship

The skeleton had:

```c
#cgo LDFLAGS: -L. -lCASBACnetStack_x64_Debug
```

Two problems: it names the **Debug** configuration (this example only ever
builds/ships **Release** - see the top-level README.md "Why only a Release
build"), and `-L.` resolves relative to the **build's working directory**,
not anywhere predictable at run time.

Fixed to:

```c
#cgo windows LDFLAGS: -L../lib -lCASBACnetStack_x64_Release
#cgo linux LDFLAGS: -L../lib -lCASBACnetStack_x64_Release
```

**Chosen layout:** the built native library is placed in `lib/` at this
repository's root (relative to `cas_bacnet_stack/`, hence `-L../lib`) for
the **build-time link**, and then copied next to the built executable for
the **run-time load** (the OS's dynamic loader does not consult `#cgo
LDFLAGS` - that only tells the linker what to embed at build time; see the
top-level README.md "Put the native library where the built binary can
find it").

## Already correct - no fix needed

- **`AddNetworkPortObject`** is already the folded single-call signature in
  the header
  (`BACnetStack_AddNetworkPortObject(deviceInstance, objectInstance,
  networkType, protocolLevel, networkNumber, networkNumberQuality,
  referencePort)`), and cgo picks it up correctly as-is (verified: this
  repository's `AddNetworkPortObject` Go wrapper calls it directly, no
  adaptation needed).
- **The `errorCode` out-param** is present on every `Get*Property`
  callback in the header, and every trampoline here includes and threads
  it through (`property_dispatch.go`'s `getPropertyCharString` is the one
  place this example actually sets it - an invalid `State_Text` array
  index - matching the Rust sibling's one deliberate use of the error
  code).

## Toolchain finding: the header-include path references the submodule directly

`cas_bacnet_stack_adapter.go`'s `#cgo CFLAGS: -I../submodules/cas-bacnet-stack/source`
points the C preprocessor at the submodule's own copy of
`CASBACnetStackDLL.h` rather than a second, hand-copied header living in
this directory. This is a deliberate, narrow exception to "never reference
the submodule in place": the *adapter* (this directory's `.go`/`.c` files)
is vendored and genuinely modified, but the *header* is 482 KB of
Chipkin-owned C declarations this example does not - and must not - alter a
single byte of. Copying it verbatim would only create a second file that
silently drifts from the submodule on the next `6.x` update; a read-only
`-I` reference keeps it exactly in sync and makes clear no part of the
header itself was "fixed" (there was nothing wrong with it - only the
adapter's use of it was stale).

## Toolchain findings, verified while building this example (not assumed)

- **Go 1.26.3's cgo function-value representation** (see Fix #2 above) -
  `C.fpCallbackXxx` used as a bare value needs an explicit
  `(*[0]byte)(unsafe.Pointer(...))` conversion on every registration to
  link, not just where the skeleton already had one.
- **The cgo preamble comment is compiled twice** - once for the `.go`
  file's own object, again inside the generated `_cgo_export.c` - so a
  non-`static` function *definition* placed directly in the preamble
  becomes a duplicate-symbol link error (`multiple definition of
  'fpCallbackReceiveMessageForPort'`), and marking it `static` instead
  breaks the *other* direction (the definition becomes invisible to the
  translation unit that needs to take its address for registration). The
  fix is `cas_bacnet_stack_trampolines.c`: a real, separately-compiled `.c`
  file (cgo picks up any `.c` file in the package directory automatically),
  defined exactly once, with ordinary external linkage. See that file's own
  header comment for the full mechanism.
- **This example's own `go build` in CI (see `.github/workflows/release.yml`)
  is, as far as this project is aware, the first automated compilation gate
  the Go adapter has ever had** against the current `6.x` header - it was
  silently broken (see Fix #1-#4 above). Treat any *future* header change
  as unverified again for this Go adapter until CI (or another automated Go
  build) re-checks it, the same caution the Rust sibling's `README.md`
  documents for its own adapter.
