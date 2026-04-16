package packval

import (
	"encoding/json"
	"fmt"
	"strings"
)

func FormatResult(result *Result, format string) (string, error) {
	switch strings.TrimSpace(format) {
	case "", "json":
		out, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(out), nil
	case "text":
		var b strings.Builder
		b.WriteString(fmt.Sprintf("status: %s\n", result.Status))
		for _, p := range result.Phases {
			b.WriteString(fmt.Sprintf("- %s: %s\n", p.Phase, p.Status))
			if p.Reason != "" {
				b.WriteString(fmt.Sprintf("  reason: %s\n", p.Reason))
			}
		}
		for _, e := range result.Errors {
			b.WriteString(fmt.Sprintf("ERROR [%s/%s] %s\n", e.Phase, e.Check, e.Message))
		}
		for _, w := range result.Warnings {
			b.WriteString(fmt.Sprintf("WARN [%s/%s] %s\n", w.Phase, w.Check, w.Message))
		}
		return b.String(), nil
	default:
		return "", fmt.Errorf("unknown format %q", format)
	}
}
