package adapters

import "testing"

func TestQuoteMssqlLiteral(t *testing.T) {
	tests := map[string]string{
		`top_secret-123!`: `N'top_secret-123!'`,
		`it's`:            `N'it''s'`,
		`back\slash`:      `N'back\slash'`,
		``:                `N''`,
	}

	for input, expected := range tests {
		if got := quoteMssqlLiteral(input); got != expected {
			t.Errorf("quoteMssqlLiteral(%q) = %q, want %q", input, got, expected)
		}
	}
}
