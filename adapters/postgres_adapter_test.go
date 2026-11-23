package adapters_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
	"github.com/jackc/pgx/v5"
)

func preparePostgresDB(t *testing.T) (context.Context, adapters.DatabaseAdapter, ClientConnectTest) {
	databaseHost := "localhost"
	databasePort := "5432"

	ctx := context.Background()
	adminURL := fmt.Sprintf("postgres://%s:%s@%s:%s?sslmode=disable", "postgres", "pA_sw0rd", databaseHost, databasePort)
	adapter, err := adapters.GetPostgresConnection(ctx, adminURL)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(ctx context.Context, databaseName string, databaseUsername string, databasePassword string) error {
		dsn := url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(databaseUsername, databasePassword),
			Host:     net.JoinHostPort(databaseHost, databasePort),
			Path:     databaseName,
			RawQuery: "sslmode=disable",
		}
		client, err := pgx.Connect(ctx, dsn.String())
		if err != nil {
			return err
		}
		defer client.Close(ctx)

		_, err = client.Exec(ctx, "CREATE TABLE test (id int);")
		return err
	}

	return ctx, adapter, clientConnectTest
}

func TestPostgresDB(t *testing.T) {
	ctx, adapter, clientConnectTest := preparePostgresDB(t)

	testHelper(t, ctx, adapter, clientConnectTest)
}

func TestPostgresDBCleanup(t *testing.T) {
	ctx, adapter, _ := preparePostgresDB(t)

	cleanupTestHelper(t, ctx, adapter)
}

// TestPostgresRepairsUserWithoutSchemaPrivileges simulates a user created by an older version of
// the operator on PostgreSQL >= 15 (without privileges on the public schema) and checks that it is
// detected and repaired, and that CreateDatabaseUser can safely be called multiple times.
func TestPostgresRepairsUserWithoutSchemaPrivileges(t *testing.T) {
	ctx, adapter, clientConnectTest := preparePostgresDB(t)
	databaseName := "repair-db_123"
	databaseUsername := "repair-user_123"
	databasePassword := "top_secret-123!"

	admin, err := pgx.Connect(ctx, "postgres://postgres:pA_sw0rd@localhost:5432?sslmode=disable")
	if err != nil {
		t.Fatalf("Error opening admin connection: %s", err)
	}

	t.Cleanup(func() {
		if err := adapter.DeleteDatabaseUser(ctx, databaseName, databaseUsername); err != nil {
			t.Errorf("Error deleting database user: %s", err)
		}
		if err := adapter.DeleteDatabase(ctx, databaseName); err != nil {
			t.Errorf("Error deleting database: %s", err)
		}
		admin.Close(ctx)
		adapter.Close(ctx)
	})

	// given: a user created the old way (only privileges on the database)
	if err := adapter.CreateDatabase(ctx, databaseName); err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	for _, query := range []string{
		fmt.Sprintf("CREATE USER \"%s\" WITH PASSWORD '%s';", databaseUsername, databasePassword),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";", databaseName, databaseUsername),
	} {
		if _, err := admin.Exec(ctx, query); err != nil {
			t.Fatalf("Error creating legacy user: %s", err)
		}
	}
	if err := clientConnectTest(ctx, databaseName, databaseUsername, databasePassword); err == nil {
		t.Fatalf("Expected legacy user to be unable to create tables")
	}

	// then: the missing privileges are detected
	hasAccess, err := adapter.HasDatabaseUserWithAccess(ctx, databaseName, databaseUsername)
	if err != nil {
		t.Fatalf("Error checking user access: %s", err)
	}
	if hasAccess {
		t.Fatalf("Expected legacy user to be reported as missing access")
	}

	// when: the reconciler calls CreateDatabaseUser (twice to check idempotency)
	for i := 0; i < 2; i++ {
		if err := adapter.CreateDatabaseUser(ctx, databaseName, databaseUsername, databasePassword); err != nil {
			t.Fatalf("Error repairing database user (run %d): %s", i+1, err)
		}
	}

	// then: the user has full access
	hasAccess, err = adapter.HasDatabaseUserWithAccess(ctx, databaseName, databaseUsername)
	if err != nil {
		t.Fatalf("Error checking user access: %s", err)
	}
	if !hasAccess {
		t.Fatalf("Expected repaired user to have access")
	}
	if err := clientConnectTest(ctx, databaseName, databaseUsername, databasePassword); err != nil {
		t.Fatalf("Error using database with repaired user: %s", err)
	}
}

func TestPostgresEnsureExtensions(t *testing.T) {
	ctx, adapter, _ := preparePostgresDB(t)
	databaseName := "extension-db_123"
	databaseUsername := "extension-user_123"
	databasePassword := "top_secret-123!"

	extensionAdapter, ok := adapter.(adapters.ExtensionAdapter)
	if !ok {
		t.Fatalf("Expected postgres adapter to support extensions")
	}

	t.Cleanup(func() {
		if err := adapter.DeleteDatabaseUser(ctx, databaseName, databaseUsername); err != nil {
			t.Errorf("Error deleting database user: %s", err)
		}
		if err := adapter.DeleteDatabase(ctx, databaseName); err != nil {
			t.Errorf("Error deleting database: %s", err)
		}
		adapter.Close(ctx)
	})

	if err := adapter.CreateDatabase(ctx, databaseName); err != nil {
		t.Fatalf("Error creating database: %s", err)
	}
	if err := adapter.CreateDatabaseUser(ctx, databaseName, databaseUsername, databasePassword); err != nil {
		t.Fatalf("Error creating database user: %s", err)
	}

	// vector is not a trusted extension, so the database user can't create it on its own
	client, err := pgx.Connect(ctx, fmt.Sprintf("postgres://%s:%s@localhost:5432/%s?sslmode=disable", databaseUsername, url.QueryEscape(databasePassword), databaseName))
	if err != nil {
		t.Fatalf("Error connecting as database user: %s", err)
	}
	defer client.Close(ctx)
	if _, err := client.Exec(ctx, "CREATE EXTENSION vector;"); err == nil {
		t.Fatalf("Expected database user to be unable to create the vector extension")
	}

	// called twice to check idempotency
	for i := 0; i < 2; i++ {
		if err := extensionAdapter.EnsureExtensions(ctx, databaseName, []string{"vector", "pg_trgm"}); err != nil {
			t.Fatalf("Error ensuring extensions (run %d): %s", i+1, err)
		}
	}

	// the database user can use the extensions
	if _, err := client.Exec(ctx, "CREATE TABLE items (id int, embedding vector(3));"); err != nil {
		t.Fatalf("Error creating table with vector column: %s", err)
	}
	if _, err := client.Exec(ctx, "INSERT INTO items VALUES (1, '[1,2,3]'), (2, '[4,5,6]');"); err != nil {
		t.Fatalf("Error inserting vectors: %s", err)
	}
	var nearest int
	if err := client.QueryRow(ctx, "SELECT id FROM items ORDER BY embedding <-> '[3,1,2]' LIMIT 1;").Scan(&nearest); err != nil {
		t.Fatalf("Error querying vectors: %s", err)
	}
	if nearest != 1 {
		t.Fatalf("Expected nearest item to be 1, got %d", nearest)
	}

	// extensions which aren't installed on the server are reported
	err = extensionAdapter.EnsureExtensions(ctx, databaseName, []string{"does_not_exist"})
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("Expected error for missing extension, got: %v", err)
	}

	// invalid extension names are rejected
	if err := extensionAdapter.EnsureExtensions(ctx, databaseName, []string{`vector"; DROP DATABASE postgres; --`}); err == nil {
		t.Fatalf("Expected error for invalid extension name")
	}
}
