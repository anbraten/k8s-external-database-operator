package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type postgresAdapter struct {
	db *pgx.Conn
}

func (adapter postgresAdapter) HasDatabase(ctx context.Context, database string) (bool, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM pg_database WHERE datname='%s'", database)
	err := adapter.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

func (adapter postgresAdapter) CreateDatabase(ctx context.Context, database string) error {
	query := fmt.Sprintf("CREATE DATABASE \"%s\";", database)
	_, err := adapter.db.Exec(ctx, query)
	return err
}

func (adapter postgresAdapter) DeleteDatabase(ctx context.Context, database string) error {
	query := fmt.Sprintf("DROP DATABASE \"%s\";", database)
	_, err := adapter.db.Exec(ctx, query)
	return err
}

// HasDatabaseUserWithAccess checks that the user exists and has all privileges granted by
// CreateDatabaseUser. Checking the schema privileges (and not only CONNECT, which is granted to
// PUBLIC by default) ensures that users missing them, e.g. users created by older versions of
// this operator on PostgreSQL >= 15, are detected and repaired by CreateDatabaseUser.
func (adapter postgresAdapter) HasDatabaseUserWithAccess(ctx context.Context, database string, username string) (bool, error) {
	roleExists, err := adapter.roleExists(ctx, username)
	if err != nil || !roleExists {
		return false, err
	}

	dbExists, err := adapter.HasDatabase(ctx, database)
	if err != nil || !dbExists {
		return false, err
	}

	var hasDatabasePrivileges bool
	err = adapter.db.QueryRow(ctx,
		"SELECT has_database_privilege($1, $2, 'CONNECT') AND has_database_privilege($1, $2, 'CREATE') AND has_database_privilege($1, $2, 'TEMPORARY');",
		username, database).Scan(&hasDatabasePrivileges)
	if err != nil || !hasDatabasePrivileges {
		return false, err
	}

	// schema privileges can only be checked from inside the database
	conn, err := adapter.connectToDatabase(ctx, database)
	if err != nil {
		return false, err
	}
	defer conn.Close(ctx)

	var hasSchemaPrivileges bool
	err = conn.QueryRow(ctx,
		"SELECT has_schema_privilege($1, 'public', 'USAGE') AND has_schema_privilege($1, 'public', 'CREATE');",
		username).Scan(&hasSchemaPrivileges)
	if err != nil {
		return false, err
	}

	return hasSchemaPrivileges, nil
}

// CreateDatabaseUser creates the user (if it doesn't exist yet) and grants it all privileges on
// the database. All steps are idempotent, so a partially failed run is completed by the next
// reconcile and existing users with missing privileges are repaired.
// The password of an already existing user is not changed.
func (adapter postgresAdapter) CreateDatabaseUser(ctx context.Context, database string, username string, password string) error {
	roleExists, err := adapter.roleExists(ctx, username)
	if err != nil {
		return err
	}

	if !roleExists {
		// make password sql safe
		quotedPassword := quotePostgresLiteral(password)
		query := fmt.Sprintf("CREATE USER \"%s\" WITH PASSWORD %s", username, quotedPassword)
		_, err = adapter.db.Exec(ctx, query)
		if err != nil {
			return err
		}
	}

	query := fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";", database, username)
	_, err = adapter.db.Exec(ctx, query)
	if err != nil {
		return err
	}

	// Since PostgreSQL 15 the CREATE privilege on the public schema is no longer granted to
	// all users by default, so it has to be granted explicitly inside the new database.
	conn, err := adapter.connectToDatabase(ctx, database)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	query = fmt.Sprintf("GRANT ALL ON SCHEMA public TO \"%s\";", username)
	_, err = conn.Exec(ctx, query)

	return err
}

func (adapter postgresAdapter) DeleteDatabaseUser(ctx context.Context, database string, username string) error {
	conn, err := adapter.connectToDatabase(ctx, database)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	// Drops all objects owned by the user in the database and revokes all privileges granted to it
	// on objects in this database and on shared objects (like the database itself).
	query := fmt.Sprintf("DROP OWNED BY \"%s\";", username)
	_, err = conn.Exec(ctx, query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf("DROP USER \"%s\";", username)
	_, err = adapter.db.Exec(ctx, query)
	return err
}

// EnsureExtensions creates the given extensions in the database. Extensions are created by the admin
// user of the operator, as most extensions (e.g. vector) are not trusted and therefore can't be
// created by the database user itself.
func (adapter postgresAdapter) EnsureExtensions(ctx context.Context, database string, extensions []string) error {
	if len(extensions) == 0 {
		return nil
	}

	conn, err := adapter.connectToDatabase(ctx, database)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	for _, extension := range extensions {
		if err := IsValidIdentifier(extension); err != nil {
			return err
		}

		var exists bool
		err = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_extension WHERE extname = $1);", extension).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		var available bool
		err = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = $1);", extension).Scan(&available)
		if err != nil {
			return err
		}
		if !available {
			return fmt.Errorf("extension '%s' is not installed on the postgres server", extension)
		}

		query := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS \"%s\";", extension)
		if _, err = conn.Exec(ctx, query); err != nil {
			return err
		}
	}

	return nil
}

func (adapter postgresAdapter) roleExists(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := adapter.db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1);", username).Scan(&exists)
	return exists, err
}

// connectToDatabase opens a new connection with the same settings (host, credentials, TLS, ...)
// as the admin connection, but to the given database.
func (adapter postgresAdapter) connectToDatabase(ctx context.Context, database string) (*pgx.Conn, error) {
	config := adapter.db.Config().Copy()
	config.Database = database
	return pgx.ConnectConfig(ctx, config)
}

// quotePostgresLiteral quotes a string literal to be used inside a PostgreSQL statement.
// Single quotes are doubled. If the literal contains backslashes they are
// escaped as well and the literal is prefixed with E (postgres escape string syntax).
// Behaves the same as pq.QuoteLiteral from github.com/lib/pq.
func quotePostgresLiteral(literal string) string {
	literal = strings.ReplaceAll(literal, `'`, `''`)
	if strings.Contains(literal, `\`) {
		literal = strings.ReplaceAll(literal, `\`, `\\`)
		return ` E'` + literal + `'`
	}

	return `'` + literal + `'`
}

func (adapter postgresAdapter) Close(ctx context.Context) error {
	return adapter.db.Close(ctx)
}

func GetPostgresConnection(ctx context.Context, url string) (*postgresAdapter, error) {
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return nil, err
	}

	adapter := postgresAdapter{
		db: conn,
	}

	return &adapter, nil
}
