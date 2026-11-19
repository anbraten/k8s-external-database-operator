package adapters

import (
	"fmt"
	"regexp"
	"strings"
)

var IdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9]+?[a-zA-Z0-9_-]*?$`)

// QuoteLiteral quotes a string literal to be used inside an SQL statement.
// Single quotes are doubled. If the literal contains backslashes they are
// escaped as well and the literal is prefixed with E (postgres escape string syntax).
// Behaves the same as pq.QuoteLiteral from github.com/lib/pq.
func QuoteLiteral(literal string) string {
	literal = strings.ReplaceAll(literal, `'`, `''`)
	if strings.Contains(literal, `\`) {
		literal = strings.ReplaceAll(literal, `\`, `\\`)
		return ` E'` + literal + `'`
	}

	return `'` + literal + `'`
}

func IsValidIdentifier(txt string) error {
	if IdentifierRegex.MatchString(txt) {
		return nil
	}

	return fmt.Errorf("'%s' is not an allowed identifier. Please make sure it matches '^[a-zA-Z0-9]+?[a-zA-Z0-9_-]*?$'", txt)
}
