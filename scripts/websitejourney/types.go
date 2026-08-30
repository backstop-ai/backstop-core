package main

type DualIdentity struct {
	JourneyLinkID string `yaml:"journey_link_id"`
	BoundaryID    string `yaml:"boundary_id"`
	Href          string `yaml:"href"`
}

type GeneratedDescriptor struct {
	Kind          string `yaml:"kind"`
	CommitBinding string `yaml:"commit_binding"`
	Path          string `yaml:"path,omitempty"`
	Commit        string `yaml:"commit,omitempty"`
}

type MappedObligation struct {
	Kind                  string                `yaml:"kind"`
	ClaimID               string                `yaml:"claim_id,omitempty"`
	OwnerRoute            string                `yaml:"owner_route,omitempty"`
	OwnerAnchor           string                `yaml:"owner_anchor,omitempty"`
	ClaimType             string                `yaml:"claim_type,omitempty"`
	BoundaryID            string                `yaml:"boundary_id,omitempty"`
	State                 string                `yaml:"state,omitempty"`
	JobID                 string                `yaml:"job_id,omitempty"`
	Output                string                `yaml:"output,omitempty"`
	TruthBeginMarker      string                `yaml:"truth_begin_marker,omitempty"`
	TruthEndMarker        string                `yaml:"truth_end_marker,omitempty"`
	SourcesBeginMarker    string                `yaml:"sources_begin_marker,omitempty"`
	SourcesEndMarker      string                `yaml:"sources_end_marker,omitempty"`
	SourceDigest          string                `yaml:"source_digest,omitempty"`
	Descriptors           []GeneratedDescriptor `yaml:"descriptors,omitempty"`
	URLTemplates          []string              `yaml:"url_templates,omitempty"`
	SiteIdentity          string                `yaml:"site_identity,omitempty"`
	RenderedAnchors       []string              `yaml:"rendered_anchors,omitempty"`
	ReconstructionVerdict string                `yaml:"reconstruction_verdict,omitempty"`
	RenderedDigest        string                `yaml:"rendered_digest,omitempty"`
	CopiedProse           bool                  `yaml:"copied_prose,omitempty"`
	Inferred              bool                  `yaml:"inferred,omitempty"`
	Seed5Resolved         bool                  `yaml:"seed5_resolved,omitempty"`
	ReconstructEnvelope   bool                  `yaml:"reconstruct_envelope,omitempty"`
}

type MappedJourney struct {
	GlobalKey    string             `yaml:"global_key"`
	Hops         []string           `yaml:"hops"`
	JLinks       []string           `yaml:"jlinks"`
	Obligations  []MappedObligation `yaml:"obligations"`
	DualIdentity DualIdentity       `yaml:"dual_identity"`
}

type MapPrerequisite struct {
	ID      string `yaml:"id"`
	Spec    string `yaml:"spec"`
	Command string `yaml:"command"`
}

type WebsiteCapabilityMap struct {
	Root                 string            `yaml:"-"`
	SchemaVersion        string            `yaml:"schema_version"`
	PredecessorVersions  map[string]string `yaml:"predecessor_versions"`
	Prerequisites        []MapPrerequisite `yaml:"prerequisites"`
	AdoptionInstructions []string          `yaml:"adoption_instructions"`
	Journeys             []MappedJourney   `yaml:"journeys"`
}

type GeneratedObligation struct {
	JobID                 string
	SourceDigest          string
	TruthBeginMarker      string
	TruthEndMarker        string
	SourcesBeginMarker    string
	SourcesEndMarker      string
	Descriptors           []GeneratedDescriptor
	URLTemplates          []string
	SiteIdentity          string
	RenderedAnchors       []string
	ReconstructionVerdict string
	RenderedDigest        string
	ReconstructEnvelope   bool
}

func (m WebsiteCapabilityMap) PrerequisiteIDs() []string {
	ids := make([]string, 0, len(m.Prerequisites))
	for _, prerequisite := range m.Prerequisites {
		ids = append(ids, prerequisite.ID)
	}
	return ids
}

func (m WebsiteCapabilityMap) AdoptionInstructionIDs() []string {
	return append([]string(nil), m.AdoptionInstructions...)
}

func (m WebsiteCapabilityMap) Journey(key string) MappedJourney {
	for _, journey := range m.Journeys {
		if journey.GlobalKey == key {
			return journey
		}
	}
	return MappedJourney{GlobalKey: key}
}

func (j MappedJourney) GeneratedJobIDs() []string {
	var ids []string
	for _, obligation := range j.Obligations {
		if obligation.Kind == "generated" && obligation.JobID != "" {
			ids = append(ids, obligation.JobID)
		}
	}
	return ids
}

func (j MappedJourney) GeneratedObligations() []GeneratedObligation {
	var jobs []GeneratedObligation
	for _, obligation := range j.Obligations {
		if obligation.Kind != "generated" {
			continue
		}
		jobs = append(jobs, GeneratedObligation{
			JobID:                 obligation.JobID,
			SourceDigest:          obligation.SourceDigest,
			TruthBeginMarker:      obligation.TruthBeginMarker,
			TruthEndMarker:        obligation.TruthEndMarker,
			SourcesBeginMarker:    obligation.SourcesBeginMarker,
			SourcesEndMarker:      obligation.SourcesEndMarker,
			Descriptors:           append([]GeneratedDescriptor(nil), obligation.Descriptors...),
			URLTemplates:          append([]string(nil), obligation.URLTemplates...),
			SiteIdentity:          obligation.SiteIdentity,
			RenderedAnchors:       append([]string(nil), obligation.RenderedAnchors...),
			ReconstructionVerdict: obligation.ReconstructionVerdict,
			RenderedDigest:        obligation.RenderedDigest,
			ReconstructEnvelope:   obligation.ReconstructEnvelope,
		})
	}
	return jobs
}

type MapBindingMutation struct {
	Name                   string   `yaml:"name"`
	OmitKey                string   `yaml:"omit_key"`
	ExtraKey               string   `yaml:"extra_key"`
	Key                    string   `yaml:"key"`
	OverrideKind           string   `yaml:"override_kind"`
	CopyClaimStatement     bool     `yaml:"copy_claim_statement"`
	InferBoundaryFromProse bool     `yaml:"infer_boundary_from_prose"`
	OverrideClaimType      string   `yaml:"override_claim_type"`
	OverrideBoundaryState  string   `yaml:"override_boundary_state"`
	OverrideSpec           string   `yaml:"override_spec"`
	OverrideVersion        string   `yaml:"override_version"`
	ResolveSiteCommit      bool     `yaml:"resolve_site_commit"`
	OmitAdoption           string   `yaml:"omit_adoption"`
	OverrideJobs           []string `yaml:"override_jobs"`
	OmitJob                string   `yaml:"omit_job"`
	ExtraJob               string   `yaml:"extra_job"`
	ReconstructEnvelope    bool     `yaml:"reconstruct_envelope"`
	OverrideDigest         string   `yaml:"override_digest"`
	DropMarkers            bool     `yaml:"drop_markers"`
	ResolveURLTemplate     bool     `yaml:"resolve_url_template"`
	SplitDualIdentity      bool     `yaml:"split_dual_identity"`
	ExpectedError          string   `yaml:"expected_error"`
}
