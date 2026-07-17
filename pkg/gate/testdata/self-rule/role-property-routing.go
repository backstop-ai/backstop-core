package spine

// Positive fixture for Family B7 (no-rule-id-keyed-routing, ISSUE-064 REQ-004): finding
// routing on the DECLARED role carried in the finding's structured Properties (the
// ISSUE-062 channel), not a baked rule-id/engine-key literal. This is the correct,
// name-independent source — a pack may name its rules anything. Nothing here may trigger
// the rule.

type violation struct {
	Rule       string
	Properties map[string]string
}

// routeByRole partitions findings by the pack-DECLARED substantiveness_role property. The
// rule NAME is never consulted; only the declared role decides the partition.
func routeByRole(vs []violation) (hollow, extraction []violation) {
	for _, v := range vs {
		switch v.Properties["substantiveness_role"] {
		case "hollow":
			hollow = append(hollow, v)
		case "referenced-symbol":
			extraction = append(extraction, v)
		}
	}
	return hollow, extraction
}
