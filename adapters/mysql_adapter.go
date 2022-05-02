package adapters

import (
	"context"
	"database/sql"
	"fmt"

	// load SQL driver for mysql
	_ "github.com/go-sql-driver/mysql"
)

type mysqlAdapter struct {
	db *sql.DB
}

func (adapter mysqlAdapter) HasDatabase(ctx context.Context, database string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME=?"
	err := adapter.db.QueryRowContext(ctx, query, database).Scan(&count)
	return count == 1, err
}

func (adapter mysqlAdapter) CreateDatabase(ctx context.Context, database string) error {
	query := fmt.Sprintf("CREATE DATABASE %s;", database)
	_, err := adapter.db.ExecContext(ctx, query)
	return err
}

func (adapter mysqlAdapter) DeleteDatabase(ctx context.Context, database string) error {
	query := fmt.Sprintf("DROP DATABASE %s;", database)
	_, err := adapter.db.ExecContext(ctx, query)
	return err
}

func (adapter mysqlAdapter) HasDatabaseUserWithAccess(ctx context.Context, database string, username string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM mysql.db WHERE Db=? AND USER=?"
	err := adapter.db.QueryRowContext(ctx, query, database, username).Scan(&count)
	return count == 1, err
}

func (adapter mysqlAdapter) CreateDatabaseUser(ctx context.Context, database string, username string, password string) error {
	// make password sql safe
	quotedPassword := QuoteLiteral(password)
	query := fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY %s;", username, quotedPassword)
	_, err := adapter.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	query = fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%%';", database, username)
	_, err = adapter.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}

	_, err = adapter.db.ExecContext(ctx, "FLUSH PRIVILEGES;")
	return err
}

func (adapter mysqlAdapter) DeleteDatabaseUser(ctx context.Context, database string, username string) error {
	query := fmt.Sprintf("DROP USER %s;", username)
	_, err := adapter.db.ExecContext(ctx, query)
	return err
}

func (adapter mysqlAdapter) Close(ctx context.Context) error {
	return adapter.db.Close()
}

func GetMysqlConnection(ctx context.Context, dsn string) (*mysqlAdapter, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	adapter := mysqlAdapter{
		db: db,
	}

	return &adapter, nil
}
