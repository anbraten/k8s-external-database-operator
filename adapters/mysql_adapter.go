package adapters

import (
	"database/sql"

	// SQL driver for mysql
	_ "github.com/go-sql-driver/mysql"
)

type mysqlAdapter struct {
	db *sql.DB
}

func (adapter mysqlAdapter) HasDatabase(database string) (bool, error) {
	return false, nil
}

func (adapter mysqlAdapter) CreateDatabase(name string) error {
	_, err := adapter.db.Exec("CREATE DATABASE IF NOT EXISTS $1;", name)
	return err
}

func (adapter mysqlAdapter) DeleteDatabase(name string) error {
	_, err := adapter.db.Exec("DROP DATABASE IF EXISTS $1;", name)
	return err
}

func (adapter mysqlAdapter) HasDatabaseUserWithAccess(username string, database string) (bool, error) {
	// TODO implement
	return false, nil
}

func (adapter mysqlAdapter) CreateDatabaseUser(username string, password string, database string) error {
	// TODO implement
	return nil
}

func (adapter mysqlAdapter) DeleteDatabaseUser(database string, username string) error {
	// TODO implement
	return nil
}

func (adapter mysqlAdapter) Close() error {
	return adapter.db.Close()
}

func GetMysqlConnection(host string, adminUsername string, adminPassword string) (*mysqlAdapter, error) {
	db, err := sql.Open("mysql", adminUsername+":"+adminPassword+"@"+host)
	if err != nil {
		return nil, err
	}

	adapter := mysqlAdapter{
		db: db,
	}

	return &adapter, nil
}
