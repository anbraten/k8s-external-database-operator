package adapters_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
	"github.com/go-kivik/kivik/v4"
)

func prepareCouchDB(t *testing.T) (context.Context, adapters.DatabaseAdapter, ClientConnectTest) {
	databaseHost := "localhost"
	databasePort := "5984"

	ctx := context.Background()
	adminURL := fmt.Sprintf("http://%s:%s@%s:%s", "admin", "pA_sw0rd", databaseHost, databasePort)
	adapter, err := adapters.GetCouchdbConnection(ctx, adminURL)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(ctx context.Context, databaseName string, databaseUsername string, databasePassword string) error {
		dsn := url.URL{
			Scheme: "http",
			User:   url.UserPassword(databaseUsername, databasePassword),
			Host:   net.JoinHostPort(databaseHost, databasePort),
		}
		client, err := kivik.New("couch", dsn.String())
		if err != nil {
			return err
		}
		defer client.Close()

		_, _, err = client.DB(databaseName).CreateDoc(ctx, map[string]interface{}{"test": "test"})
		return err
	}

	return ctx, adapter, clientConnectTest
}

func TestCouchDB(t *testing.T) {
	ctx, adapter, clientConnectTest := prepareCouchDB(t)

	testHelper(t, ctx, adapter, clientConnectTest)
}

func TestCouchDBCleanup(t *testing.T) {
	ctx, adapter, _ := prepareCouchDB(t)

	cleanupTestHelper(t, ctx, adapter)
}
