package pack

import (
	"fmt"
	"regexp"
	"strings"
)

// ParseCoordinate parses a versioned coordinate reference.
func ParseCoordinate(ref string) (*Coordinate, error) {
	parts := strings.Split(ref, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid coordinate format")
	}

	packParts := strings.Split(parts[0], "@")
	if len(packParts) != 2 || packParts[0] == "" || packParts[1] == "" {
		return nil, fmt.Errorf("invalid coordinate pack segment")
	}

	itemParts := strings.Split(parts[1], "@")
	if len(itemParts) == 0 || len(itemParts) > 2 || itemParts[0] == "" {
		return nil, fmt.Errorf("invalid coordinate item segment")
	}
	if len(itemParts) == 2 && itemParts[1] == "" {
		return nil, fmt.Errorf("invalid coordinate item version")
	}

	coord := &Coordinate{
		PackName:    packParts[0],
		PackVersion: packParts[1],
		ItemName:    itemParts[0],
	}
	if len(itemParts) == 2 {
		coord.ItemVersion = itemParts[1]
	}

	return coord, nil
}

// NamespacedRuleID prefixes a rule ID with the pack name.
func NamespacedRuleID(packName, ruleID string) string {
	return packName + "/" + ruleID
}

// ValidateRuleID validates lowercase kebab-case rule IDs.
func ValidateRuleID(id string) error {
	if !ruleIDPattern.MatchString(id) {
		return fmt.Errorf("rule ID must match %s", ruleIDPattern.String())
	}
	return nil
}

var ruleIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)
