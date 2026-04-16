package packval

type Result struct {
	Status   string              `json:"status"`
	Pack     string              `json:"pack"`
	Version  string              `json:"version"`
	Phases   []PhaseResult       `json:"phases"`
	Errors   []ValidationError   `json:"errors"`
	Warnings []ValidationWarning `json:"warnings"`
}

type PhaseResult struct {
	Phase      string              `json:"phase"`
	Status     string              `json:"status"`
	Checks     int                 `json:"checks"`
	DurationMs int64               `json:"duration_ms"`
	Reason     string              `json:"reason,omitempty"`
	Errors     []ValidationError   `json:"errors,omitempty"`
	Warnings   []ValidationWarning `json:"warnings,omitempty"`
}

type ValidationError struct {
	Phase        string `json:"phase"`
	Check        string `json:"check"`
	Rule         string `json:"rule,omitempty"`
	Claim        string `json:"claim,omitempty"`
	Message      string `json:"message"`
	FixHint      string `json:"fix_hint,omitempty"`
	ManifestPath string `json:"manifest_path,omitempty"`
}

type ValidationWarning struct {
	Phase   string   `json:"phase"`
	Check   string   `json:"check"`
	Message string   `json:"message"`
	Files   []string `json:"files,omitempty"`
	FixHint string   `json:"fix_hint,omitempty"`
}

func (r *Result) FinalizeStatus() {
	if len(r.Errors) == 0 {
		r.Status = "pass"
		return
	}
	r.Status = "fail"
}
