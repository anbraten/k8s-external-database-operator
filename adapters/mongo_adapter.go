package adapters

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoAdapter struct {
	client *mongo.Client
}

func (adapter mongoAdapter) HasDatabase(ctx context.Context, database string) (bool, error) {
	databaseNames, err := adapter.client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		return false, err
	}

	return contains(databaseNames, database), err
}

func (adapter mongoAdapter) CreateDatabase(ctx context.Context, database string) error {
	// create dummy data as mongo only creates databases if they contain something
	_, err := adapter.client.Database(database).Collection("delete_me").InsertOne(ctx, bson.D{{Key: "empty", Value: true}})
	return err
}

func (adapter mongoAdapter) DeleteDatabase(ctx context.Context, database string) error {
	return adapter.client.Database(database).Drop(ctx)
}

func (adapter mongoAdapter) HasDatabaseUserWithAccess(ctx context.Context, database string, username string) (bool, error) {
	var result bson.Raw

	command := bson.D{{Key: "usersInfo", Value: bson.M{"user": username, "db": database}}}
	err := adapter.client.Database("admin").RunCommand(ctx, command).Decode(&result)
	if err != nil {
		return false, err
	}

	usersArr, ok := result.Lookup("users").ArrayOK()
	if !ok {
		return false, errors.New("cant find users array in result")
	}

	users, err := usersArr.Elements()
	if err != nil {
		return false, err
	}

	return len(users) == 1, nil
}

func (adapter mongoAdapter) CreateDatabaseUser(ctx context.Context, database string, username string, password string) error {
	return adapter.client.Database(database).RunCommand(
		ctx,
		bson.D{
			{Key: "createUser", Value: username},
			{Key: "pwd", Value: password},
			{Key: "roles", Value: []bson.M{{"role": "dbAdmin", "db": database}}}}).Err()
}

func (adapter mongoAdapter) DeleteDatabaseUser(ctx context.Context, database string, username string) error {
	return adapter.client.Database(database).RunCommand(ctx, bson.D{{Key: "dropUser", Value: username}}).Err()
}

func (adapter mongoAdapter) Close(ctx context.Context) error {
	return adapter.client.Disconnect(ctx)
}

func GetMongoConnection(ctx context.Context, url string) (*mongoAdapter, error) {
	clientOpts := options.Client().ApplyURI(url)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	adapter := mongoAdapter{
		client: client,
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
