package adapters

import "errors"

type DatabaseAdapter interface {
	CreateDatabase(name string) error
	DeleteDatabase(name string) error
	UpdateDatabaseUser(username string, password string) error
	Close() error
}

func CreateConnection(databaseType string, host string, adminUsername string, adminPassword string) (DatabaseAdapter, error) {
	if databaseType == "mysql" {
		return createMysql(host, adminUsername, adminPassword)
	}

	if databaseType == "mysql" {
		return createCouchdb(host, adminUsername, adminPassword)
	}

	return nil, errors.New("Can't find database adapter")
}
