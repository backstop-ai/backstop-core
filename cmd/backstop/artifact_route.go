package main

import (
	"fmt"
	"strings"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
)

// ValidatorFunc is the signature for type-specific validator functions
// in pkg/validate.
type ValidatorFunc func(*artifact.ParsedArtifact, *schema.Schema) validate.ValidationResult

// validatorRouter maps schema_version prefixes to their validator functions.
var validatorRouter = map[string]ValidatorFunc{
	"spec":      validate.Spec,
	"plan":      validate.Plan,
	"adr":       validate.ADR,
	"bundle":    validate.Bundle,
	"issue":     validate.Issue,
	"directive": validate.Directive,
}

// RouteValidator extracts the schema_version from the artifact metadata and
// returns the appropriate validator function. Returns an error if the
// schema_version is missing or has an unrecognized prefix.
func RouteValidator(art *artifact.ParsedArtifact) (ValidatorFunc, error) {
	version := ""
	for k, v := range art.Metadata {
		if strings.EqualFold(k, "schema_version") || strings.EqualFold(k, "schema-version") {
			version = v
			break
		}
	}

	if version == "" {
		return nil, fmt.Errorf("missing schema_version field")
	}

	parts := strings.SplitN(version, "/", 2)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid schema_version format: %q", version)
	}

	prefix := parts[0]
	fn, ok := validatorRouter[prefix]
	if !ok {
		return nil, fmt.Errorf("unrecognized schema_version prefix: %q", prefix)
	}

	return fn, nil
}
