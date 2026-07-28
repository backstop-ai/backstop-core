---
name: scratch-module-probes-production
description: During a TDD-red phase you can exercise real production code without touching the repo — a scratch Go module with a `replace` directive back into backstop-core
metadata:
  type: project
---

In the RED phase you own only test files, so you cannot drop a temporary `main`
or probe test into the repo to check that a fixture is well-formed. Build a
throwaway module in the scratchpad instead:

```
module probe
go 1.23
require github.com/bmanson/backstop-core v0.0.0
replace github.com/bmanson/backstop-core => /Users/bmanson/src/projects/backstop-core
```

`GOFLAGS=-mod=mod go mod tidy && go run .` then calls any EXPORTED production
API (`recipe.ParseRecipeManifest`, `packval.NewPipeline(...).Run()`) against a
fixture. Zero repo files written, no gate scope shifted, no sibling-agent
collision.

Pair it with a **swap probe** when the fixture cannot pass yet because the
production branch it needs does not exist: copy the fixture to a temp dir, `sed`
the one blocking field to a value the CURRENT code accepts (e.g.
`archetype: recipes` -> `archetype: code`), and run the pipeline. Every phase
except the deliberately-swapped one is then proven green, so you hand off
knowing the only thing between the fixture and a pass is the branch the next
task adds — instead of discovering a second latent fixture defect a phase later.

**Why:** the red phase otherwise proves only "it fails", never "it fails for
exactly one reason". **How to apply:** any phase where fixtures land before the
code that consumes them.

Related: [[project_hermetic_pack_fixture_recipe]],
[[project_minimal_valid_pack_fixture]].
