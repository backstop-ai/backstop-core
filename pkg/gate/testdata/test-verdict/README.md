# test-verdict fixture

`go-test-failure.sarif.json` is DERIVED, not captured and not hand-written.

Regenerate with:

    /bin/sh cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/scripts/test-to-sarif.sh \
      < cmd/backstop/testdata/go-toolchain/fixtures/go-test-failures.txt \
      > pkg/gate/testdata/test-verdict/go-test-failure.sarif.json

Generated 2026-08-16 by exactly that command.

The SOURCE of truth is the `cmd/backstop` capture — real `go test` stdout for
package `github.com/example/project/pkg/widget`. This file is derived from it by
the committed go-toolchain converter, so an edit to EITHER the capture or the
converter must regenerate this file. Never edit this file by hand.

Two properties make it load-bearing for the mandated-test verdict join:

1. BARE-BASENAME URIs. The failures report `widget_test.go` / `gadget_test.go`
   for a package whose import path is `.../pkg/widget`, so the reported path can
   never match a gate scope's canonicalized repo-relative file set.
2. ONE RESULT WITH NO LOCATION AT ALL. `TestNoPos` failed with no `file:line`
   block, so the converter emits an empty `uri` and `startLine: 0`.

Together they are why the join is keyed on the TEST NAME rather than the path.
