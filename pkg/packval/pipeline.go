package packval

import (
	"fmt"
	"path/filepath"
	"time"
)

type Pipeline struct {
	packDir string
	opts    PipelineOptions
}

type PipelineOptions struct {
	Mode     string
	Format   string
	Executor FixtureExecutor
}

func NewPipeline(packDir string, opts PipelineOptions) *Pipeline {
	return &Pipeline{packDir: packDir, opts: opts}
}

func (p *Pipeline) Run() *Result {
	out := &Result{Phases: []PhaseResult{}, Errors: []ValidationError{}, Warnings: []ValidationWarning{}}
	manifestPath := filepath.Join(p.packDir, "pack.yml")
	m, err := ParseManifest(manifestPath)
	if err != nil {
		p1 := PhaseResult{
			Phase:  "phase1-structural",
			Status: "fail",
			Errors: []ValidationError{{Phase: "phase1-structural", Check: "parse-manifest", Message: err.Error(), ManifestPath: "pack.yml"}},
		}
		out.Phases = append(out.Phases, p1)
		out.Errors = append(out.Errors, p1.Errors...)
		skipList := []string{"phase2-coherence", "phase3-fixtures", "phase4-archetype", "phase5-layer", "phase6-risk-class"}
		for _, phase := range skipList {
			if p.opts.Mode == "check" && phase == "phase3-fixtures" {
				continue
			}
			out.Phases = append(out.Phases, PhaseResult{Phase: phase, Status: "skipped", Reason: "phase1 failed"})
		}
		out.FinalizeStatus()
		return out
	}
	out.Pack = m.Name
	out.Version = m.Version

	phases := []struct {
		name string
		run  func() *PhaseResult
	}{
		{"phase1-structural", func() *PhaseResult { return RunStructural(m, p.packDir) }},
		{"phase2-coherence", func() *PhaseResult { return RunCoherence(m, p.packDir) }},
	}
	if p.opts.Mode == "test" {
		phases = append(phases, struct {
			name string
			run  func() *PhaseResult
		}{"phase3-fixtures", func() *PhaseResult { return RunFixtures(m, p.packDir, p.opts.Executor) }})
	}
	phases = append(phases,
		struct {
			name string
			run  func() *PhaseResult
		}{"phase4-archetype", func() *PhaseResult { return RunArchetype(m, p.packDir) }},
		struct {
			name string
			run  func() *PhaseResult
		}{"phase5-layer", func() *PhaseResult { return RunLayer(m, p.packDir) }},
		struct {
			name string
			run  func() *PhaseResult
		}{"phase6-risk-class", func() *PhaseResult { return RunRiskClass(m) }},
	)

	stopped := false
	failedPhase := ""
	for _, phase := range phases {
		if stopped {
			out.Phases = append(out.Phases, PhaseResult{Phase: phase.name, Status: "skipped", Reason: fmt.Sprintf("%s failed", failedPhase)})
			continue
		}
		start := time.Now()
		pr := phase.run()
		pr.DurationMs = time.Since(start).Milliseconds()
		out.Phases = append(out.Phases, *pr)
		out.Errors = append(out.Errors, pr.Errors...)
		out.Warnings = append(out.Warnings, pr.Warnings...)
		if pr.Status == "fail" {
			stopped = true
			failedPhase = phase.name
		}
	}
	// Check mode legitimately produces no phase3-fixtures entry; the phase list above
	// simply never appends it. A dead scan that computed that absence and did nothing
	// with it used to sit here.
	out.FinalizeStatus()
	return out
}
