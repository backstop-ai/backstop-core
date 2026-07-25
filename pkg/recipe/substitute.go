package recipe

import (
	"fmt"
	"strings"
)

// The placeholder delimiters. A span between them is a PARAM NAME and nothing
// else — there is no expression grammar to open.
const (
	placeholderOpen  = "{{"
	placeholderClose = "}}"
)

// Substitute resolves {{ param }} placeholders in template from params.
//
// This is pure value interpolation and is deliberately NOT Turing-complete
// (REQ-002): the span between the delimiters is trimmed and looked up as a param
// NAME. There is no expression parser, no conditional, no loop, no pipeline, and
// no function map, so a logic construct written inside a placeholder is never
// evaluated as code — it is simply a name no param declares, and it fails loud.
// Substituted values are never rescanned, so a value that itself contains the
// delimiters cannot smuggle a second pass in.
//
// An unresolvable name returns an error naming the placeholder and NO string: a
// partially substituted or silently blanked result is never returned, because it
// would look applied while being malformed.
//
// A `{{` with no matching `}}` is not a placeholder; it is copied through as the
// literal text it is.
func Substitute(template string, params map[string]string) (string, error) {
	var out strings.Builder
	rest := template

	for {
		open := strings.Index(rest, placeholderOpen)
		if open < 0 {
			out.WriteString(rest)
			break
		}

		body := rest[open+len(placeholderOpen):]
		end := strings.Index(body, placeholderClose)
		if end < 0 {
			out.WriteString(rest)
			break
		}

		out.WriteString(rest[:open])

		name := strings.TrimSpace(body[:end])
		value, declared := params[name]
		if !declared {
			return "", fmt.Errorf("unresolvable placeholder %s %s %s: no declared param named %q", placeholderOpen, name, placeholderClose, name)
		}
		out.WriteString(value)

		rest = body[end+len(placeholderClose):]
	}

	return out.String(), nil
}
