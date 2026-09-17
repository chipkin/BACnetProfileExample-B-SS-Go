# Changelog - `common/` (Go edition)

All notable changes to the vendored Go `common/` helpers. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [1.0.0] - 2026-09-16

### Added

- `simple_udp.go` - application-owned UDP socket (`net.ListenUDP`,
  `net.UDPConn`), non-blocking in effect via a short per-call read
  deadline. Written from scratch: no transport helper exists anywhere in
  `submodules/cas-bacnet-stack/adapters/golang/` (it ships
  `CASBACnetStackAdapter.go` and `propertybufferhelper/` only), matching
  the Rust edition's own "no transport helper to vendor" precedent.
  Poll-based (`Recv()` calls `ReadFromUDP` directly once per stack tick),
  matching the C++/C#/Python/Rust editions' single-threaded model.
- `cas_example_helper.go` - `GetLocalIPv4()` (a UDP-dial probe + assumed
  `/24` netmask, matching the Python/Rust editions' documented deviation -
  Go's standard library can enumerate interfaces but picking the "primary"
  one reliably needs a route-table lookup that is itself
  platform-specific) and CLI argument parsing helpers (`--port`,
  `--deviceID`). **Deviation from the C#/Python/Rust editions of this
  file:** `register_common_callbacks()`/`send_i_am()`/`print_version()` do
  **not** live here - Go's cgo requires every direct `C.BACnetStack_*` call
  and every `//export`ed callback to live in the one package with `import
  "C"`, so those pieces live in `cas_bacnet_stack/` instead, and `main.go`
  calls them directly. See this file's own header comment for the full
  explanation.
- `constants.go` - the BACnet enumeration values this example needs, plus
  this example's own device/object model constants (device name, instance
  numbers, colour names, Multi-State Input state text). Every enumeration
  value matches the BACnet standard and the C++/C#/Python/Rust editions'
  equivalent files 1:1 - ported fresh here (like the Rust edition) because
  the Go adapter, like the Rust one, is a pure callback-binding surface
  with no enumeration constants of its own.
- `device_state.go` - new file (present in the Rust edition too, for the
  same reason; absent from the C++/C#/Python editions, whose
  callback-registration APIs accept closures/delegates that can capture
  `main()`-local state directly). A package-level `DeviceState` value,
  guarded by a `sync.Mutex` and accessed only through `WithState()`, holds
  every piece of mutable device state a `Get*Property` callback needs to
  read. Necessary because cgo `//export` functions must be top-level,
  non-method, non-closure functions - a hard cgo rule (not a style choice,
  unlike Rust's `extern "C" fn` restriction, which is a language-level
  function-pointer-type restriction reaching the same conclusion by a
  different route). See the file's own doc comment for the full
  explanation; this is the single biggest structural difference this
  edition shares with the Rust one, and the reason `cas_bacnet_stack/`
  imports `common` rather than the other way around.

### First Go `common/` in this series - what makes it different from the C++/C#/Python/Rust editions

This is the first Go `common/` in the BACnet profile example series. The
systematic differences versus the other four editions (all targeting the
same `submodules/cas-bacnet-stack` `6.x` branch and the same callback API -
trailing `errorCode` out-param, folded `AddNetworkPortObject()`, `*ForPort`
transport callbacks keyed by Network Port instance - none of that is new
here):

- **Build-time linking (cgo), not runtime dynamic loading.** Unlike C#'s
  implicit P/Invoke resolution, Python's `ctypes.CDLL()`, or the Rust
  edition's explicit `libloading::Library` lazy-load, cgo links the native
  library at **build time** via `#cgo LDFLAGS` (see
  `cas_bacnet_stack/README.md` fix #5). There is no "native library missing"
  exception to catch in Go code - a missing DLL/so fails the OS loader
  before `main()` runs at all, with a native OS error message.
- **A real vendored adapter that needed real fixes** (see
  `cas_bacnet_stack/README.md`) - unlike the C#/Python/Rust adapters, which
  are pure FFI binding surfaces that never shipped broken, the Go adapter
  skeleton had genuinely stopped compiling against the current header.
- **Package-level state for the same underlying reason as Rust, enforced
  by a different mechanism.** cgo's "exported functions must be top-level"
  rule and Rust's "`extern "C" fn` cannot capture" rule both rule out
  closures, but for different underlying reasons (a linker/ABI requirement
  vs. a language-level function-pointer type). The practical result -
  `device_state.go`/`device_state.rs` - is the same shape either way.
- **`common/` does not own callback registration**, unlike every other
  edition's `CASExampleHelper` equivalent - see `cas_example_helper.go`'s
  deviation note above.
