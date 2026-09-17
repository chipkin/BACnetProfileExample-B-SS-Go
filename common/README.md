# `common/` - cgo-free shared plumbing

Everything in this package is **ordinary Go with no cgo dependency** -
`main.go` and `cas_bacnet_stack/` both import it, but it never imports
`cas_bacnet_stack` itself (that would be a cgo dependency creeping into a
package that does not need one). See `CHANGELOG.md` for what's here and why
it looks different from the C#/Python/Rust editions' `CASExampleHelper`
equivalent, and `device_state.go`'s own doc comment for the single biggest
structural reason (package-level state, not closures).

| File | What it is |
|---|---|
| `constants.go` | BACnet enumeration values + this example's device/object model constants |
| `device_state.go` | the device's mutable state - package-level, mutex-guarded, read/written through `WithState()` |
| `simple_udp.go` | an application-owned, non-blocking UDP socket |
| `cas_example_helper.go` | local IPv4 discovery + CLI argument parsing (see its header comment for what is deliberately **not** here) |
