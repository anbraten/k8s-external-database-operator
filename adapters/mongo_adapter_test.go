package adapters_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func prepareMongoDB(t *testing.T) (context.Context, adapters.DatabaseAdapter, ClientConnectTest) {
	databaseHost := "localhost"
	databasePort := "27017"

	ctx := context.Background()
	adminURL := fmt.Sprintf("mongodb://%s:%s@%s:%s/?authSource=admin", "admin", "pA_sw0rd", databaseHost, databasePort)
	adapter, err := adapters.GetMongoConnection(ctx, adminURL)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(ctx context.Context, databaseName string, databaseUsername string, databasePassword string) error {
		dsn := url.URL{
			Scheme: "mongodb",
			User:   url.UserPassword(databaseUsername, databasePassword),
			Host:   net.JoinHostPort(databaseHost, databasePort),
			Path:   databaseName,
		}
		clientOpts := options.Client().ApplyURI(dsn.String())
		client, err := mongo.Connect(clientOpts)
		if err != nil {
			return err
		}
		defer client.Disconnect(ctx)

		_, err = client.Database(databaseName).Collection("test").InsertOne(ctx, map[string]interface{}{"test": "test"})
		return err
	}

	return ctx, adapter, clientConnectTest
}

func TestMongoDB(t *testing.T) {
	ctx, adapter, clientConnectTest := prepareMongoDB(t)

	testHelper(t, ctx, adapter, clientConnectTest)
}

func TestMongoDBCleanup(t *testing.T) {
	ctx, adapter, _ := prepareMongoDB(t)

	cleanupTestHelper(t, ctx, adapter)
}
