package adapters

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoAdapter struct {
	client  *mongo.Client
	context context.Context
}

func (adapter mongoAdapter) HasDatabase(database string) (bool, error) {
	databaseNames, err := adapter.client.ListDatabaseNames(adapter.context, bson.D{{Key: "empty", Value: false}})

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
	return adapter.client.Database(name).Drop(adapter.context)
}

func (adapter mongoAdapter) HasDatabaseUserWithAccess(username string, database string) (bool, error) {
	// TODO implement
	return false, nil
}

func (adapter mongoAdapter) UpdateDatabaseUser(username string, password string, database string) error {
	r := adapter.client.Database(database).RunCommand(
		adapter.context,
		bson.D{
			{Key: "createUser", Value: username},
			{Key: "pwd", Value: password},
			{Key: "roles", Value: []bson.M{{"role": "dbAdmin", "db": database}}}})

	if r.Err() != nil {
		return r.Err()
	}

	return nil
}

func (adapter mongoAdapter) Close() error {
	return adapter.client.Disconnect(context.TODO())
}

func GetMongoConnection(url string) (*mongoAdapter, error) {
	clientOpts := options.Client().ApplyURI(url)
	client, err := mongo.Connect(context.TODO(), clientOpts)

	if err != nil {
		return nil, err
	}

	adapter := mongoAdapter{
		client:  client,
		context: context.Background(),
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
