package adapters

import (
	"github.com/leesper/couchdb-golang"
)

type couchdbAdapter struct {
	db *couchdb.Server
}

func (adapter couchdbAdapter) HasDatabase(database string) (bool, error) {
	return false, nil
}

func (adapter couchdbAdapter) CreateDatabase(database string) error {
	// TODO create if not exists
	_, err := adapter.db.Create(database)
	return err
}

func (adapter couchdbAdapter) DeleteDatabase(database string) error {
	// TODO delete if exists
	return adapter.db.Delete(database)
}

func (adapter couchdbAdapter) HasDatabaseUserWithAccess(username string, database string) (bool, error) {
	// TODO implement
	return false, nil
}

func (adapter couchdbAdapter) UpdateDatabaseUser(database string, username string, password string) error {
	// TODO implement
	return nil
}

func (adapter couchdbAdapter) Close() error {
	// TODO implement
	return nil
}

func GetCouchdbConnection(url string, adminUsername string, adminPassword string) (*couchdbAdapter, error) {
	server, err := couchdb.NewServer(url)
	if err != nil {
		return nil, err
	}

	_, err = server.Login(adminUsername, adminPassword)
	if err != nil {
		return nil, err
	}

	adapter := couchdbAdapter{
		db: server,
	}

	return &adapter, nil
}
