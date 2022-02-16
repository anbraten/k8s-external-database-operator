package adapters_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMongoDB(t *testing.T) {
	databaseHost := "localhost"
	databasePort := "27017"

	ctx := context.Background()
	mongodbUrl := fmt.Sprintf("mongodb://admin:1234@%s:%s/?authSource=admin", databaseHost, databasePort)
	adapter, err := adapters.GetMongoConnection(ctx, mongodbUrl)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(databaseName string, databaseUsername string, databasePassword string) error {
		clientOpts := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s:%s/%s", databaseUsername, databasePassword, databaseHost, databasePort, databaseName))
		client, err := mongo.Connect(ctx, clientOpts)
		if err != nil {
			return err
		}
		defer client.Disconnect(ctx)
		_, err = client.Database(databaseName).Collection("test").InsertOne(ctx, map[string]interface{}{"test": "test"})
		return err
	}

	testHelper(t, ctx, adapter, clientConnectTest)
}
