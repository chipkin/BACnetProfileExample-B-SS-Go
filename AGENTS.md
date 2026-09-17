# AGENTS.md

Ground rules for an agent (or a careful human) working in this repository.

## What this project is

A tutorial BACnet B-SS (Smart Sensor) device, in Go, built on the CAS
BACnet Stack via cgo. Read-only, single Device, four objects. See
README.md for the full picture and `cas_bacnet_stack/README.md` for the
adapter fixes this example made.

## Layout

- `main.go` - CLI, device/object setup, run loop, interactive commands.
- `cas_bacnet_stack/` - the **only** package with `import "C"`. Every
  direct `C.BACnetStack_*` call and every `//export`ed callback lives here.
  Never move a cgo call or an `//export` function out of this package -
  cgo requires exports and their preamble to live together, and this
  repository's `main.go` deliberately has zero cgo surface of its own.
- `common/` - cgo-free. Device state, UDP transport, IPv4 discovery, CLI
  parsing. Never add an `import "C"` here.
- `submodules/cas-bacnet-stack/` - **never edit anything under this path.**
  It is a pinned, upstream-owned submodule (branch `6.x`). If the vendored
  adapter needs a fix, fix the **copy** in `cas_bacnet_stack/`, and record
  the fix in `cas_bacnet_stack/README.md`.
- `lib/` - where the built native library (`CASBACnetStack_x64_Release.dll`
  / `.so`) is placed for the build-time link (`#cgo LDFLAGS: -L../lib`).
  Also copy it next to the built executable for the run-time load - the
  two are separate steps (see README.md).

## Build

```sh
git submodule update --init --recursive
# build or copy the native library into lib/ - see README.md
CGO_ENABLED=1 go build -o bacnet-b-ss-go .
cp lib/CASBACnetStack_x64_Release.dll .   # next to the built binary
```

`CGO_ENABLED=1` is not optional - this repository does not build without
cgo. A C compiler must be on `PATH` for cgo itself (MinGW-w64/gcc on
Windows - separate from the MSVC toolchain that builds the native DLL).

## Run

```sh
./bacnet-b-ss-go --port 47808 --deviceID 389001
```

`--help` and `--version` both exit 0 without starting the device or
touching the network - useful for a smoke test that only needs to confirm
the binary links and the native library loads.

## Conventions

- **Package-level state, never a closure, in `cas_bacnet_stack/`.** Every
  `//export`ed function must be a top-level, non-method, non-closure
  function - a hard cgo rule. Device state lives in `common`'s
  `DeviceState`, accessed only through `common.WithState(func(s
  *DeviceState) { ... })`. Do not add a `main()`-local variable a callback
  needs to read; it has no way to reach it.
- **Write into C-supplied buffers directly, never hand C a Go slice's
  backing array that could be moved or collected.** See
  `property_dispatch.go`'s `getPropertyCharString` and
  `cas_bacnet_stack_adapter.go`'s corresponding trampoline for the pattern
  (`C.memcpy` into the caller's pointer, not `unsafe.Pointer(&goSlice[0])`
  handed across the call boundary and read back later).
- **Callback trampoline bodies belong in `cas_bacnet_stack_trampolines.c`,
  never in the cgo preamble comment.** See that file's header comment - the
  preamble compiles twice (once per `.go` file's own object, again inside
  the generated `_cgo_export.c`), so a non-`static` function *definition*
  there is a duplicate-symbol link error, and `static` breaks the address-
  of registration `RegisterCallbacks()` needs. A real `.c` file has neither
  problem.
- **Registrations use `(*[0]byte)(unsafe.Pointer(C.fpCallbackXxx))`, not a
  bare `C.fpCallbackXxx` reference.** See `cas_bacnet_stack/README.md` fix
  #2 for why the "pass it directly, no cast" ideal does not compile on the
  Go toolchain this repository was built and verified against
  (`go1.26.3`), and why the real type-safety check still holds anyway (the
  `fpCallbackXxx` C trampolines are declared with the header's exact
  parameter types).
- **Never modify anything under `submodules/`.** Fix the vendored copy in
  `cas_bacnet_stack/` instead, and document the fix in that directory's
  `README.md`.

## How to verify a change

1. `CGO_ENABLED=1 go build -o bacnet-b-ss-go .` - must be 0 errors.
2. `go vet . ./cas_bacnet_stack ./common` - should be clean. **Do not use
   `go vet ./...` or `go build ./...`** in this repository: both recurse
   into `submodules/cas-bacnet-stack/adapters/golang/`, which has its own
   (non-vendored, upstream) `.go` files with no `go.mod` boundary of their
   own to stop Go's tooling from trying to compile them as part of this
   module - and they fail (missing header include path, no `-I` set for
   that directory). This is a submodule-layout quirk, not a bug in this
   repository's own code; list packages explicitly instead.
3. Copy the native library next to the built binary; run
   `./bacnet-b-ss-go --version` - confirms the link/load path works without
   touching the network.
4. Run `./bacnet-b-ss-go --port <free port>` in the background; confirm
   the `FYI: Device ... ready.` log line appears and the process survives
   a few seconds; kill it; confirm the log shows no panic/crash.
5. If you changed a `Get*Property` callback's routing, cross-check
   `docs/objects.json` and `docs/PICS.md` still describe the same served-by
   answer for every property you touched - a half-added object does not
   fail loudly (the stack falls back to a stack-generated default with no
   error), so a routing mistake here is silent unless you check by hand or
   with a real BACnet explorer.

## Releasing

Tag-triggered: `.github/workflows/release.yml` builds the native library
and the Go binary for Windows and Linux, runs the smoke test above in CI,
and attaches both platforms' binaries (with their native library) to a
GitHub Release on a `v*` tag push. No release step runs on every push to
`main` - only on a tag.

## A note on this stack's Go adapter's CI history

`submodules/cas-bacnet-stack/adapters/golang/CASBACnetStackAdapter.go` had
**no automated compilation gate** against the current `6.x` header before
this repository's `go build` in CI - see CHANGELOG.md and
`cas_bacnet_stack/README.md` for the specifics of what was broken and how
it was fixed. Treat any *future* change to that submodule's header, or to
this repository's vendored copy of the adapter, as unverified again until
CI re-checks it.

## License

This example's own code (everything outside `submodules/`) is CC0-1.0 -
see LICENSE. The `submodules/cas-bacnet-stack` submodule is a separately
licensed Chipkin product; see its own license terms.
