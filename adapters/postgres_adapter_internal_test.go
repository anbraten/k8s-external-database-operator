package adapters

import "testing"

func TestQuotePostgresLiteral(t *testing.T) {
	tests := map[string]string{
		`top_secret-123!`: `'top_secret-123!'`,
		`it's`:            `'it''s'`,
		`back\slash`:      ` E'back\\slash'`,
		`it's\`:           ` E'it''s\\'`,
		``:                `''`,
	}

	for input, expected := range tests {
		if got := quotePostgresLiteral(input); got != expected {
			t.Errorf("quotePostgresLiteral(%q) = %q, want %q", input, got, expected)
		}
	}
}
