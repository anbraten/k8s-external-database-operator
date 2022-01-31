package adapters

import (
	"context"
	"database/sql"

	// SQL driver for mysql
	_ "github.com/go-sql-driver/mysql"
)

type mysqlAdapter struct {
	db *sql.DB
}

func (adapter mysqlAdapter) HasDatabase(ctx context.Context, database string) (bool, error) {
	return false, nil
}

func (adapter mysqlAdapter) CreateDatabase(ctx context.Context, name string) error {
	_, err := adapter.db.Exec("CREATE DATABASE IF NOT EXISTS $1;", name)
	return err
}

func (adapter mysqlAdapter) DeleteDatabase(ctx context.Context, name string) error {
	_, err := adapter.db.Exec("DROP DATABASE IF EXISTS $1;", name)
	return err
}

func (adapter mysqlAdapter) HasDatabaseUserWithAccess(ctx context.Context, username string, database string) (bool, error) {
	// TODO implement
	return false, nil
}

func (adapter mysqlAdapter) CreateDatabaseUser(ctx context.Context, username string, password string, database string) error {
	// TODO implement
	return nil
}

func (adapter mysqlAdapter) DeleteDatabaseUser(ctx context.Context, database string, username string) error {
	// TODO implement
	return nil
}

func (adapter mysqlAdapter) Close(ctx context.Context) error {
	return adapter.db.Close()
}

func GetMysqlConnection(ctx context.Context, host string, adminUsername string, adminPassword string) (*mysqlAdapter, error) {
	db, err := sql.Open("mysql", adminUsername+":"+adminPassword+"@"+host)
	if err != nil {
		return nil, err
	}

	adapter := mysqlAdapter{
		db: db,
	}

	return &adapter, nil
}
