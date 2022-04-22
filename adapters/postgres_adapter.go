package adapters

import (
	"context"
	"database/sql"
	"fmt"

	// load SQL driver for postgres
	_ "github.com/lib/pq"
)

type postgresAdapter struct {
	db *sql.DB
}

func (adapter postgresAdapter) HasDatabase(ctx context.Context, database string) (bool, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM pg_database WHERE datname=%s", database)
	err := adapter.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

func (adapter postgresAdapter) CreateDatabase(ctx context.Context, database string) error {
	query := fmt.Sprintf("CREATE DATABASE %s", database)
	_, err := adapter.db.ExecContext(ctx, query)
	return err
}

func (adapter postgresAdapter) DeleteDatabase(ctx context.Context, database string) error {
	query := fmt.Sprintf("DROP DATABASE %s", database)
	_, err := adapter.db.ExecContext(ctx, query)
	return err
}

func (adapter postgresAdapter) HasDatabaseUserWithAccess(ctx context.Context, database string, username string) (bool, error) {
	var hasPrivilege bool
	query := fmt.Sprintf("SELECT has_database_privilege(%s, %s, 'CONNECT');", username, database)
	err := adapter.db.QueryRowContext(ctx, query).Scan(&hasPrivilege)
	if err != nil {
		return false, err
	}
	return hasPrivilege, nil
}

func (adapter postgresAdapter) CreateDatabaseUser(ctx context.Context, database string, username string, password string) error {
	// make password sql safe
	quotedPassword := QuoteLiteral(password)
	query := fmt.Sprintf("CREATE USER %s WITH PASSWORD %s", username, quotedPassword)
	_, err := adapter.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", database, username)
	_, err = adapter.db.ExecContext(ctx, query)

	return err
}

func (adapter postgresAdapter) DeleteDatabaseUser(ctx context.Context, database string, username string) error {
	query := fmt.Sprintf("REVOKE ALL PRIVILEGES ON DATABASE %s FROM %s; REVOKE ALL ON SCHEMA public FROM %s;", database, username, username)
	_, err := adapter.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf("DROP OWNED BY %s; DROP USER %s;", username, username)
	_, err = adapter.db.ExecContext(ctx, query)
	return err
}

func (adapter postgresAdapter) Close(ctx context.Context) error {
	return adapter.db.Close()
}

func GetPostgresConnection(ctx context.Context, url string) (*postgresAdapter, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, err
	}

	adapter := postgresAdapter{
		db: db,
	}

	return &adapter, nil
}
