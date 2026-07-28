// Package baseengines loads the backstop-owned BASE ENGINE pack (the four generic
// engines: semgrep, ast-grep, sandbox, config-file) from an embedded pack.yml and
// projects it into an engine.Registry (ISSUE-027). It is the mechanism that lets
// the binary hold ZERO baked engine knowledge: the engine facts live as DATA in
// the embedded packs/base-engines/pack.yml, parsed through the NORMAL
// pack.ParseManifest path, and this package merely enumerates whatever that pack
// declares. It replaces the deleted engine.DefaultRegistry() / DefaultFieldContracts()
// baked tables.
//
// Import direction (no cycle): baseengines imports pkg/pack + pkg/pack/engine to
// parse and project; NOTHING in pkg/pack imports baseengines. cmd/backstop owns the
// embed and passes Registry() into both the gate path (resolveEngineRegistry) and
// the validation path (ValidateManifest, by parameter injection).
package baseengines

import (
	"fmt"
	"sync"

	backstopcore "github.com/backstop-ai/backstop-core"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// baseEnginesPath is the embedded location of the base ENGINE pack manifest inside
// backstopcore.BaseEnginesFS.
const baseEnginesPath = "packs/base-engines/pack.yml"

var (
	loadOnce sync.Once
	loaded   engine.Registry
)

// Registry returns the base engine bindings loaded from the embedded base pack:
// semgrep, ast-grep, sandbox, and config-file, each with its inline FieldContract.
// The embedded pack is parsed ONCE through pack.ParseManifest (the normal pack
// path) and cached; every call returns a fresh copy so a caller's merge can never
// mutate the shared table.
//
// It fails LOUD (panic) if the embedded pack is missing or unparseable: the pack is
// compiled into the binary, so a failure here is a build/authoring defect, never a
// runtime input error — silently returning an empty registry would resurrect the
// vacuous-green the eradication exists to kill.
func Registry() engine.Registry {
	loadOnce.Do(func() {
		data, err := backstopcore.BaseEnginesFS.ReadFile(baseEnginesPath)
		if err != nil {
			panic(fmt.Sprintf("baseengines: read embedded %s: %v", baseEnginesPath, err))
		}
		reg, err := registryFromBytes(data)
		if err != nil {
			panic(fmt.Sprintf("baseengines: load embedded base pack: %v", err))
		}
		loaded = reg
	})

	out := make(engine.Registry, len(loaded))
	for name, binding := range loaded {
		out[name] = binding
	}
	return out
}

// registryFromBytes parses a base-engine pack.yml through pack.ParseManifest (the
// normal pack path) and projects the parsed manifest's Engines map into an
// engine.Registry. It is engine-BLIND: it enumerates whatever the pack declares. It
// fails loud on unparseable bytes or a pack that declares no engines — the embedded
// pack is compiled in, so those are authoring defects, never silent empties.
func registryFromBytes(data []byte) (engine.Registry, error) {
	m, err := pack.ParseManifest(data)
	if err != nil {
		return nil, fmt.Errorf("parse embedded %s: %w", baseEnginesPath, err)
	}

	if len(m.Engines) == 0 {
		return nil, fmt.Errorf("embedded base pack %s declares no engines", baseEnginesPath)
	}

	reg := make(engine.Registry, len(m.Engines))
	for name, spec := range m.Engines {
		reg[name] = spec.Binding
	}
	return reg, nil
}
