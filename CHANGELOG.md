# Changelog

All notable changes to this example. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [1.0.0] - 2026-09-16

First Go implementation of the BACnet B-SS (Smart Sensor) profile in this
series, pinned to CAS BACnet Stack `6.x` at commit
`ec60c71801aad076a46c0ec783c5566a6cd0f3a0` (tag `v5.3.4-711-gec60c718`) -
the same commit the Rust and Python siblings already built against, so the
prebuilt `CASBACnetStack_x64_Release.dll` was **reused, not rebuilt**
(copied from the Rust sibling's `target/release/`).

### Added

- Full device/object model: Device 389001 "Rainbow" (vendor 389, Chipkin),
  Analog Input 1 "Bronze", Binary Input 1 "Emerald", Multi-State Input 1
  "Hot Pink", Network Port 1 "Vermilion" - see `docs/objects.json` and
  `docs/PICS.md`.
- `cas_bacnet_stack/` - the vendored + fixed Go cgo adapter. See
  `cas_bacnet_stack/README.md` for the complete, itemized list of fixes.
  In short: the vendored skeleton
  (`submodules/cas-bacnet-stack/adapters/golang/CASBACnetStackAdapter.go`)
  **did not compile** against the current `6.x` header - two callbacks had
  been renamed/widened (`*ForPort`), one extern declaration used the wrong
  C type (`time_t` vs. `CASBACnetTime`), and 4 of the 6 `Get*Property`
  callbacks this device model needs were never wired at all. All fixed in
  this repository's own vendored copy only; `submodules/` was never
  touched.
- `common/` - cgo-free shared plumbing (device state, UDP transport, local
  IPv4 discovery, CLI parsing). See `common/CHANGELOG.md`.
- `main.go` - CLI (`--port`, `--deviceID`, `--version`, `--help`), device/
  object setup, the run loop (`BACnetStack_Tick()` + a channel-drained
  interactive-command reader), and graceful Ctrl+C shutdown.
- `docs/objects.json`, `docs/PICS.md` - ported from the C++/Rust siblings,
  content identical (this is the same device model), language notes
  adjusted for Go.
- `.github/workflows/release.yml` - CI: checks out the submodule, builds
  the native library (MSBuild on Windows / g++ on Linux) or reuses a
  pinned-commit match, builds the Go binary with `CGO_ENABLED=1`, runs a
  smoke test (`--port <free port>`, poll for the "ready" log line, then
  kill it), and releases on tag push.

### Verified findings (not assumed - see `cas_bacnet_stack/README.md` for detail)

- **This is, as far as this project is aware, the first automated
  compilation gate the CAS BACnet Stack's Go adapter has ever had** against
  the current `6.x` header. It was silently broken (see "Added" above);
  dead-golang allowlist entries #1641/IFC-027 are tracked upstream but not
  fixed there. This example's `go build` in CI is that first gate.
- **Go 1.26.3's cgo function-value representation** required every
  `BACnetStack_RegisterCallback*` call to use an explicit
  `(*[0]byte)(unsafe.Pointer(...))` conversion, not just the calls the
  skeleton already cast that way - a toolchain-level finding made while
  building this example, documented in full in
  `cas_bacnet_stack/README.md` fix #2.
- **The cgo preamble comment compiles twice** (once for the `.go` file's
  own object, again inside the generated `_cgo_export.c`), so this example
  moved the callback trampoline *definitions* into a real, separately
  compiled `cas_bacnet_stack_trampolines.c` rather than leaving them as
  function bodies inside the preamble comment - see that file's header
  comment and `cas_bacnet_stack/README.md`'s "toolchain findings" section.

- **`go build ./...` / `go vet ./...` recurse into the submodule and fail**
  (`submodules/cas-bacnet-stack/adapters/golang/CASBACnetStackAdapter.go`
  has no `go.mod` of its own to stop Go's tooling from treating it as part
  of this module, and it needs an `-I` include path this repository never
  sets for it). Not a bug in this repository's own code - a submodule-
  layout quirk. Use explicit package paths (`go build .`, `go vet .
  ./cas_bacnet_stack ./common`) instead - see AGENTS.md.

### Native library

- Toolchain on the build machine: Go `go1.26.3`, MinGW-w64 gcc (cgo's C
  compiler), MSBuild (Visual Studio 2022 Build Tools, for the native
  library) - all already present; nothing needed installing.
- `go build` result: **success, 0 errors**, after the fixes above.
- Smoke test: `bacnet-b-ss-go.exe --port 47836`, native library copied next
  to the built executable. Printed version
  (`CAS BACnet Stack v6.0.21.0`), bound the UDP socket, logged
  `FYI: Device 389001 ("Rainbow") ready. Vendor ID 389. Type 'h' + Enter for help.`,
  stayed running for the observation window, and was killed cleanly. A
  single benign stack log line about `BACnetDataLinkSC` ("UUID has not been
  set") appears on every start - this is the stack's BACnet/SC
  sub-component noting it was not configured, harmless for a BACnet/IP-only
  device (the C#/Python/Rust siblings' own smoke-test logs show the
  same line).
