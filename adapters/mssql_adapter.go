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
	query := fmt.Sprintf("SELECT COUNT(*) FROM master.sys.databases WHERE name='%s';", database)
	err := adapter.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

func (adapter mssqlAdapter) CreateDatabase(ctx context.Context, database string) error {
	query := fmt.Sprintf("EXEC ('sp_configure ''contained database authentication'', 1; reconfigure;');")
	_, err := adapter.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf("CREATE DATABASE [%s] CONTAINMENT=PARTIAL;", database)
	_, err = adapter.db.ExecContext(ctx, query)
	return err
}

func (adapter mssqlAdapter) DeleteDatabase(ctx context.Context, database string) error {
	query := fmt.Sprintf("DROP DATABASE [%s];", database)
	_, err := adapter.db.ExecContext(ctx, query)
	return err
}

func (adapter mssqlAdapter) HasDatabaseUserWithAccess(ctx context.Context, database string, username string) (bool, error) {
	var count int
	query := fmt.Sprintf("USE [%s]; SELECT COUNT(*) FROM sys.database_principals WHERE authentication_type=2 AND name='%s';", database, username)
	err := adapter.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

func (adapter mssqlAdapter) CreateDatabaseUser(ctx context.Context, database string, username string, password string) error {
	// make password sql safe
	quotedPassword := QuoteLiteral(password)
	query := fmt.Sprintf("USE [%s]; CREATE USER [%s] WITH PASSWORD=%s", database, username, quotedPassword)
	_, err := adapter.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf("USE [%s]; ALTER ROLE db_owner ADD MEMBER [%s];", database, username)
	_, err = adapter.db.ExecContext(ctx, query)
	return err
}

func (adapter mssqlAdapter) DeleteDatabaseUser(ctx context.Context, database string, username string) error {
	query := fmt.Sprintf("USE [%s]; DROP USER %s;", database, username)
	_, err := adapter.db.ExecContext(ctx, query)
	return err
}

func (adapter mssqlAdapter) Close(ctx context.Context) error {
	return adapter.db.Close()
}

func GetMssqlConnection(ctx context.Context, url string) (*mssqlAdapter, error) {
	db, err := sql.Open("sqlserver", url)
	if err != nil {
		return nil, err
	}

	adapter := mssqlAdapter{
		db: db,
	}

	if err := adapter.db.PingContext(ctx); err != nil {
		return nil, err
	}

	return &adapter, nil
}
