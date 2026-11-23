package adapters

import "testing"

func TestQuoteMysqlLiteral(t *testing.T) {
	tests := map[string]string{
		`top_secret-123!`: `'top_secret-123!'`,
		`it's`:            `'it''s'`,
		`back\slash`:      `'back\\slash'`,
		`it's\`:           `'it''s\\'`,
		`\'`:              `'\\'''`,
		``:                `''`,
	}

	for input, expected := range tests {
		if got := quoteMysqlLiteral(input, false); got != expected {
			t.Errorf("quoteMysqlLiteral(%q, false) = %q, want %q", input, got, expected)
		}
	}

	// with NO_BACKSLASH_ESCAPES backslashes are ordinary characters
	if got := quoteMysqlLiteral(`it's\`, true); got != `'it''s\'` {
		t.Errorf("quoteMysqlLiteral(%q, true) = %q, want %q", `it's\`, got, `'it''s\'`)
	}
}
