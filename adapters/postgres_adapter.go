package adapters

import (
	"context"
	"fmt"

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

func (adapter postgresAdapter) HasDatabaseUserWithAccess(ctx context.Context, database string, username string) (bool, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM pg_roles WHERE rolname='%s';", username)
	err := adapter.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}

	var hasPrivilege bool
	query = fmt.Sprintf("SELECT has_database_privilege('%s', '%s', 'CONNECT');", username, database)
	err = adapter.db.QueryRow(ctx, query).Scan(&hasPrivilege)
	if err != nil {
		return false, err
	}

	return hasPrivilege, nil
}

func (adapter postgresAdapter) CreateDatabaseUser(ctx context.Context, database string, username string, password string) error {
	// make password sql safe
	quotedPassword := QuoteLiteral(password)
	query := fmt.Sprintf("CREATE USER \"%s\" WITH PASSWORD %s", username, quotedPassword)
	_, err := adapter.db.Exec(ctx, query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\";", database, username)
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

	query := fmt.Sprintf("DROP OWNED BY \"%s\";", username)
	_, err = conn.Exec(ctx, query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE \"%s\" FROM \"%s\"; REVOKE ALL ON SCHEMA public FROM \"%s\";", database, username, username)
	_, err = adapter.db.Exec(ctx, query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf("DROP USER \"%s\";", username)
	_, err = adapter.db.Exec(ctx, query)
	return err
}

// connectToDatabase opens a new connection with the same settings (host, credentials, TLS, ...)
// as the admin connection, but to the given database.
func (adapter postgresAdapter) connectToDatabase(ctx context.Context, database string) (*pgx.Conn, error) {
	config := adapter.db.Config().Copy()
	config.Database = database
	return pgx.ConnectConfig(ctx, config)
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
