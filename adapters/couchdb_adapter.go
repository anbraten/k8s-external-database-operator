package adapters

import (
	"fmt"

	"github.com/leesper/couchdb-golang"
)

type securityConfig struct{
	names []string
	roles []string
}

type couchdbAdapter struct {
	db *couchdb.Server
}

func (adapter couchdbAdapter) HasDatabase(database string) (bool, error) {
	return adapter.db.Contains(database), nil
}

func (adapter couchdbAdapter) CreateDatabase(database string) error {
	_, err := adapter.db.Create(database)
	return err
}

func (adapter couchdbAdapter) DeleteDatabase(database string) error {
	return adapter.db.Delete(database)
}

func (adapter couchdbAdapter) getDatabaseAdmins(database string) ([]string, error) {
	db, err := adapter.db.Get(database)
	if err != nil {
		return nil, err
	}

	sc, err := db.GetSecurity()
	if err != nil {
		return nil, err
	}

	admins, ok := sc["admins"].(securityConfig)
	if !ok {
		return nil, fmt.Errorf("can't find admin users in security context")
	}

	return admins.names, nil
}

func (adapter couchdbAdapter) HasDatabaseUserWithAccess(username string, database string) (bool, error) {
	admins, err := adapter.getDatabaseAdmins(database)
	if err != nil {
		return false, err
	}

	for _, name := range admins {
		if name == username {
			return true, nil
		}
	}

	return false, nil
}

func (adapter couchdbAdapter) CreateDatabaseUser(database string, username string, password string) error {
	adapter.db.AddUser(username, password, nil)
	db, err := adapter.db.Get(database)
	if err != nil {
		return err
	}

	sc, err := db.GetSecurity()
	if err != nil {
		return err
	}

	admins, err := adapter.getDatabaseAdmins(database)
	if err != nil {
		return err
	}

	sc["admins"] = &securityConfig{
		names: append(admins, username),
		roles: nil,
	}

	return db.SetSecurity(sc)
}

func (adapter couchdbAdapter) DeleteDatabaseUser(database string, username string) error {
	return adapter.db.RemoveUser(username)
}

func (adapter couchdbAdapter) Close() error {
	// couchdb is using single http calls and has no active connection we could close
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
