package adapters

import "errors"

type DatabaseAdapter interface {
	HasDatabase(database string) (error, bool)
	CreateDatabase(database string) error
	DeleteDatabase(database string) error
	HasDatabaseUserWithAccess(username string, database string) (error, bool)
	UpdateDatabaseUser(username string, password string, database string) error
	Close() error
}

func CreateConnection(databaseType string, host string, adminUsername string, adminPassword string) (DatabaseAdapter, error) {
	if databaseType == "mysql" {
		return createMysql(host, adminUsername, adminPassword)
	}

	if databaseType == "couchdb" {
		return createCouchdb(host, adminUsername, adminPassword)
	}

	return nil, errors.New("Can't find database adapter")
}
