package adapters

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoAdapter struct {
	host          string
	adminUsername string
	adminPassword string
	client        *mongo.Client
}

func (adapter mongoAdapter) HasDatabase(database string) (bool, error) {
	databaseNames, err := adapter.client.ListDatabaseNames(context.TODO(), bson.D{{"empty", false}})

	if err != nil {
		return false, err
	}

	return contains(databaseNames, database), err
}

func (adapter mongoAdapter) CreateDatabase(name string) error {
	adapter.client.Database(name)
	return nil
}

func (adapter mongoAdapter) DeleteDatabase(name string) error {
	return adapter.client.Database(name).Drop(context.TODO())
}

func (adapter mongoAdapter) HasDatabaseUserWithAccess(username string, database string) (bool, error) {
	// TODO implement
	return false, nil
}

func (adapter mongoAdapter) UpdateDatabaseUser(username string, password string, database string) error {
	r := adapter.client.Database(database).RunCommand(context.Background(), bson.D{{"createUser", username},
		{"pwd", password}, {"roles", []bson.M{{"role": "dbAdmin", "db": database}}}})

	if r.Err() != nil {
		return r.Err()
	}

	return nil
}

func (adapter mongoAdapter) Close() error {
	return adapter.client.Disconnect(context.TODO())
}

func createMongo(host string, adminUsername string, adminPassword string) (*mongoAdapter, error) {
	clientOpts := options.Client().ApplyURI("mongodb://" + adminUsername + ":" + adminPassword + "@" + host + ":27017")
	client, err := mongo.Connect(context.TODO(), clientOpts)

	if err != nil {
		return nil, err
	}

	adapter := mongoAdapter{
		host:          host,
		client:        client,
		adminUsername: adminUsername,
		adminPassword: adminPassword,
	}

	return &adapter, nil
}

// https://play.golang.org/p/Qg_uv_inCek
// contains checks if a string is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
