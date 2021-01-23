package adapters

import (
	"github.com/leesper/couchdb-golang"
)

type couchdbAdapter struct {
	url           string
	adminUsername string
	adminPassword string
	db            *couchdb.Server
}

func (adapter couchdbAdapter) CreateDatabase(name string) error {
	_, err := adapter.db.Create(name)
	return err
}

func (adapter couchdbAdapter) DeleteDatabase(name string) error {
	return adapter.db.Delete(name)
}

func (adapter couchdbAdapter) UpdateDatabaseUser(username string, password string) error {
	return nil
}

func (adapter couchdbAdapter) Close() error {
	return nil
}

func createCouchdb(url string, adminUsername string, adminPassword string) (*couchdbAdapter, error) {
	server, err := couchdb.NewServer(url)
	if err != nil {
		return nil, err
	}

	_, err = server.Login(adminUsername, adminPassword)
	if err != nil {
		return nil, err
	}

	adapter := couchdbAdapter{
		url:           url,
		db:            server,
		adminUsername: adminUsername,
		adminPassword: adminPassword,
	}

	return &adapter, nil
}
