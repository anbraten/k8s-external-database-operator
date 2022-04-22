package adapters

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/denisenkom/go-mssqldb"
)

type mssqlAdapter struct {
	db *sql.DB
}

func (adapter mssqlAdapter) HasDatabase(ctx context.Context, database string) (bool, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM master.sys.databases WHERE name=@p1")
	err := adapter.db.QueryRowContext(ctx, query, database).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

func (adapter mssqlAdapter) CreateDatabase(ctx context.Context, database string) error {
	_, err := adapter.db.ExecContext(ctx, "CREATE DATABASE @p1;", database)
	return err
}

func (adapter mssqlAdapter) DeleteDatabase(ctx context.Context, database string) error {
	_, err := adapter.db.ExecContext(ctx, "DROP DATABASE @p1;", database)
	return err
}

func (adapter mssqlAdapter) HasDatabaseUserWithAccess(ctx context.Context, database string, username string) (bool, error) {
	// TODO implement
	return false, nil
}

func (adapter mssqlAdapter) CreateDatabaseUser(ctx context.Context, database string, username string, password string) error {
	// make password sql safe
	quotedPassword := QuoteLiteral(password)
	query := fmt.Sprintf("CREATE USER %s WITH PASSWORD = %s", username, quotedPassword)
	_, err := adapter.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE %s TO %s;", database, username)
	_, err = adapter.db.ExecContext(ctx, query)

	return err
}

func (adapter mssqlAdapter) DeleteDatabaseUser(ctx context.Context, database string, username string) error {
	// TODO implement
	return nil
}

func (adapter mssqlAdapter) Close(ctx context.Context) error {
	return adapter.db.Close()
}

func GetMssqlConnection(ctx context.Context, url string) (*mssqlAdapter, error) {
	db, err := sql.Open("mssql", url)
	if err != nil {
		return nil, err
	}

	adapter := mssqlAdapter{
		db: db,
	}

	return &adapter, nil
}
