package adapters

import (
	"fmt"
	"regexp"
)

var IdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9]+?[a-zA-Z0-9_-]*?$`)

func IsValidIdentifier(txt string) error {
	if IdentifierRegex.MatchString(txt) {
		return nil
	}

	return fmt.Errorf("'%s' is not an allowed identifier. Please make sure it matches '^[a-zA-Z0-9]+?[a-zA-Z0-9_-]*?$'", txt)
}
