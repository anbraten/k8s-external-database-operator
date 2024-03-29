package adapters

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
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

	return err
}

func (adapter postgresAdapter) DeleteDatabaseUser(ctx context.Context, database string, username string) error {
	config := adapter.db.Config()
	rootUser := config.User
	rootPassword := config.Password
	host := config.Host
	port := config.Port
	// TODO: find prettier way to generate url from original url string
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", rootUser, rootPassword, host, port, database)
	conn, err := pgx.Connect(ctx, url)
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
