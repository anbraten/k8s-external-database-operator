package adapters_test

import (
	"context"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
)

type ClientConnectTest func(ctx context.Context, databaseName string, databaseUsername string, databasePassword string) error

func testHelper(t *testing.T, ctx context.Context, adapter adapters.DatabaseAdapter, clientConnectTest ClientConnectTest) {
	// given
	var err error
	databaseName := "guestbook-admin_123"
	databaseUsername := "guestbook-admin_123"
	databasePassword := "top_secret-123!"

	t.Cleanup(func() {
		if err = adapter.DeleteDatabaseUser(ctx, databaseName, databaseUsername); err != nil {
			t.Errorf("Error deleting database user: %s", err)
		}

		if err := adapter.DeleteDatabase(ctx, databaseName); err != nil {
			t.Errorf("Error deleting database: %s", err)
		}

		if err = adapter.Close(ctx); err != nil {
			t.Fatalf("Error closing database connection: %s", err)
		}
	})

	// when
	err = adapter.CreateDatabase(ctx, databaseName)
	if err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	err = adapter.CreateDatabaseUser(ctx, databaseName, databaseUsername, databasePassword)
	if err != nil {
		t.Fatalf("Error creating database user: %s", err)
	}

	// then
	err = clientConnectTest(ctx, databaseName, databaseUsername, databasePassword)
	if err != nil {
		t.Fatalf("Error connecting to database: %s", err)
	}

	hasDatabase, err := adapter.HasDatabase(ctx, databaseName)
	if err != nil {
		t.Fatalf("Error creating database: %s", err)
	} else if !hasDatabase {
		t.Fatalf("Database does not exists")
	}

	hasDatabaseUserWithAccess, err := adapter.HasDatabaseUserWithAccess(ctx, databaseName, databaseUsername)
	if err != nil {
		t.Fatalf("Error creating database user with access: %s", err)
	} else if !hasDatabaseUserWithAccess {
		t.Fatalf("Database user does not exists")
	}
}

func cleanupTestHelper(t *testing.T, ctx context.Context, adapter adapters.DatabaseAdapter) {
	result, err := adapter.HasDatabaseUserWithAccess(ctx, "non-existing-db", "non-existing-user")
	if err != nil {
		t.Fatalf("Checking for existing database user failed: %s", err)
	}
	if result {
		t.Fatalf("database and user existing but expecting to be non-existing")
	}
	
	// TODO: test db.DeleteDatabaseUser
	// TODO: test db.DeleteDatabase
}
