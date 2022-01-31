package adapters_test

import (
	"context"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
)

func testHelper(t *testing.T, ctx context.Context, adapter adapters.DatabaseAdapter) {
	// given
	var err error

	t.Cleanup(func() {
		if err = adapter.DeleteDatabaseUser(ctx, "guestbook", "guestbook-admin"); err != nil {
			t.Errorf("Error deleting database user: %s", err)
		}

		if err := adapter.DeleteDatabase(ctx, "guestbook"); err != nil {
			t.Errorf("Error deleting database: %s", err)
		}

		if err = adapter.Close(ctx); err != nil {
			t.Fatalf("Error closing database connection: %s", err)
		}
	})

	// when
	err = adapter.CreateDatabase(ctx, "guestbook")
	if err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	err = adapter.CreateDatabaseUser(ctx, "guestbook", "guestbook-admin", "test123")
	if err != nil {
		t.Fatalf("Error creating database user: %s", err)
	}

	// then
	res, err := adapter.HasDatabaseUserWithAccess(ctx, "guestbook", "guestbook-admin")
	if err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	if !res {
		t.Fatalf("Database user does not exists")
	}
}
