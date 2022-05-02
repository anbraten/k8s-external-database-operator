package adapters

import (
	"fmt"
	"regexp"

	"github.com/lib/pq"
)

var IdentifierRegex = regexp.MustCompile(`^[a-zA-Z0-9]+?[a-zA-Z0-9_-]*?$`)

func QuoteLiteral(txt string) string {
	// TODO: find prettier solution without needing an additional lib :cry: https://github.com/jackc/pgx/issues/868
	return pq.QuoteLiteral(txt)
}

func IsValidIdentifier(txt string) error {
	if IdentifierRegex.MatchString(txt) {
		return nil
	}

	return fmt.Errorf("'%s' is not an allowed identifier. Please make sure it matches '^[a-zA-Z0-9]+?[a-zA-Z0-9_-]*?$'", txt)
}
