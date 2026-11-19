package adapters_test

import (
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
)

func TestQuoteLiteral(t *testing.T) {
	tests := map[string]string{
		`top_secret-123!`: `'top_secret-123!'`,
		`it's`:            `'it''s'`,
		`back\slash`:      ` E'back\\slash'`,
		`it's\`:           ` E'it''s\\'`,
		``:                `''`,
	}

	for input, expected := range tests {
		if got := adapters.QuoteLiteral(input); got != expected {
			t.Errorf("QuoteLiteral(%q) = %q, want %q", input, got, expected)
		}
	}
}
