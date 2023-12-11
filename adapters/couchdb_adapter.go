package adapters

import (
	"context"

	_ "github.com/go-kivik/couchdb/v4" // The CouchDB driver
	"github.com/go-kivik/kivik/v4"
)

type securityConfig struct {
	names []string
	roles []string
}

type couchdbAdapter struct {
	db *kivik.Client
}

func (adapter couchdbAdapter) HasDatabase(ctx context.Context, database string) (bool, error) {
	return adapter.db.DBExists(ctx, database)
}

func (adapter couchdbAdapter) CreateDatabase(ctx context.Context, database string) error {
	return adapter.db.CreateDB(ctx, database)
}

func (adapter couchdbAdapter) DeleteDatabase(ctx context.Context, database string) error {
	return adapter.db.DestroyDB(ctx, database)
}

func (adapter couchdbAdapter) HasDatabaseUserWithAccess(ctx context.Context, database string, username string) (bool, error) {
	dbExists, dbExistsErr := adapter.HasDatabase(ctx, database)
	if dbExistsErr != nil {
		return false, dbExistsErr
	}
	if !dbExists {
		return false, nil
	}

	sc, err := adapter.db.DB(database).Security(ctx)
	if err != nil {
		return false, err
	}

	for _, name := range sc.Admins.Names {
		if name == username {
			return true, nil
		}
	}

	return false, nil
}

func (adapter couchdbAdapter) CreateDatabaseUser(ctx context.Context, database string, username string, password string) error {
	exists, err := adapter.db.DBExists(ctx, "_users")
	if err != nil {
		return err
	}

	if !exists {
		err := adapter.db.CreateDB(ctx, "_users")
		if err != nil {
			return err
		}
	}

	_, err = adapter.db.DB("_users").Put(ctx, kivik.UserPrefix+username, map[string]interface{}{
		"_id":      kivik.UserPrefix + username,
		"name":     username,
		"type":     "user",
		"roles":    []string{},
		"password": password,
	})
	if err != nil {
		return err
	}

	db := adapter.db.DB(database)
	sc, err := db.Security(ctx)
	if err != nil {
		return err
	}

	sc.Admins.Names = append(sc.Admins.Names, username)

	return db.SetSecurity(ctx, sc)
}

func (adapter couchdbAdapter) DeleteDatabaseUser(ctx context.Context, database string, username string) error {
	userDB := adapter.db.DB("_users")
	row := userDB.Get(ctx, kivik.UserPrefix+username)
	_, err := userDB.Delete(ctx, kivik.UserPrefix+username, row.Rev())
	return err
}

func (adapter couchdbAdapter) Close(ctx context.Context) error {
	return adapter.db.Close(ctx)
}

func GetCouchdbConnection(ctx context.Context, url string) (*couchdbAdapter, error) {
	client, err := kivik.New("couch", url)
	if err != nil {
		return nil, err
	}

	// do some call to test if connection is working
	_, err = client.ClusterStatus(ctx)
	if err != nil {
		return nil, err
	}

	adapter := couchdbAdapter{
		db: client,
	}

	return &adapter, nil
}
