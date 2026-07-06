package check

import "fmt"

// CheckType represents a validation pass type. It is the gate's neutral
// pass-identity vocabulary, stamped onto findings by the LIVE SARIF parser
// (ParsePackFindings → CheckTypeFindings). The file-extension routing manifest
// that once carried baked stack knowledge (a hard-coded extension list) was
// deleted with the in-process check engine (ISSUE-018).
type CheckType int

const (
	// CheckTypeLint runs the lint pass.
	CheckTypeLint CheckType = iota
	// CheckTypeBuild runs the build pass.
	CheckTypeBuild
	// CheckTypeTest runs the test pass.
	CheckTypeTest
	// CheckTypeFindings is the tool-neutral rule-fed findings pass (fed by a
	// pack engine such as semgrep or ast-grep). The gate-type identity is
	// neutral; the engine is a pack detail, never baked into this name.
	CheckTypeFindings
)

// String returns the string representation of a CheckType.
func (ct CheckType) String() string {
	switch ct {
	case CheckTypeLint:
		return "lint"
	case CheckTypeBuild:
		return "build"
	case CheckTypeTest:
		return "test"
	case CheckTypeFindings:
		return "findings"
	default:
		return fmt.Sprintf("unknown(%d)", int(ct))
	}
}
