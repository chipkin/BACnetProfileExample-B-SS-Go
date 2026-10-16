# TODO

Known defects outside this repository that it works around. None of them is
filed upstream yet ("not filed").

## Adapter defect, not filed: the Go adapter does not build at the Stack Pin

`submodules/cas-bacnet-stack/adapters/golang/CASBACnetStackAdapter.go` at
`64b2cfac5` (hand-maintained - `adapters/README.md` says no generator writes the
Go adapter, cgo derives the bindings from the header) cannot be used in place:

1. **It does not compile.** `RegisterBACnetStackCallbacks()` passes each C
   trampoline bare, e.g. line 160
   `C.BACnetStack_RegisterCallbackReceiveMessageForPort(C.fpCallbackReceiveMessageForPort)`
   (all ten registrations, lines 160-171). cgo types a C function used as a
   value as `unsafe.Pointer`, while the generated binding's parameter is
   `*[0]byte`. Compiled in place (as a file of a `package main` test module,
   with `-I` to `source/`, mingw gcc 13.2.0, go1.21.6):

   ```
   .\adapter.go:160:54: cannot use (_Cgo_ptr(_Cfpvar_fp_fpCallbackReceiveMessageForPort)) (value of type unsafe.Pointer) as *[0]byte value in argument to (_Cfunc_BACnetStack_RegisterCallbackReceiveMessageForPort)
   ... (the same for lines 161-171)
   ```

   The comment above those lines (#584/#2111) says the `(*[0]byte)` cast was
   removed on purpose, so cgo type-checks each registration; with this
   toolchain the bare reference does not compile at all.
2. **It is not importable.** It is `package main` (line 6) with its own example
   database (`device`, `analogInput`, the `ExampleDB*` types), no `go.mod`, and
   no `#cgo CFLAGS` include path for the `#include "CASBACnetStackDLL.h"` it
   needs (line 17) - only `#cgo LDFLAGS: -L. -lCASBACnetStack_x64_Release`
   (line 15), relative to the build directory.
3. It does not declare `goCallback*` Go functions for its `extern` declarations;
   a user has to supply all ten `//export` functions in the same package.

**Workaround (port code, marked):** `cgobridge/callbacks.go` and
`cgobridge/trampolines.c` carry the same trampoline pattern (C wrapper ->
`//export`ed Go function), registered with a `(*[0]byte)(unsafe.Pointer(...))`
cast; `cgobridge/stack.go` includes the header in place. The trampolines are
declared with the header's exact parameter types, so a header change still
fails the build. **When the adapter builds and is importable, delete
`cgobridge/callbacks.go`'s trampolines and `cgobridge/trampolines.c` and use
it.**

## Smaller notes

- Windows has no rpath: the executable finds the stack's DLL only through
  `PATH` (or a copy in the gitignored `build/`). README "Run" says so.
- The interactive commands are line-mode (key + Enter, `u`/`d` for the arrows)
  because Go's standard library cannot read raw key presses. TUTORIAL.md
  "Interactive keys are a simplification".
