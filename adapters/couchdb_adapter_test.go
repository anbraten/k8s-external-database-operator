package adapters_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
	"github.com/go-kivik/kivik/v4"
)

func TestCouchDB(t *testing.T) {
	databaseHost := "localhost"
	databasePort := "5984"

	ctx := context.Background()
	url := fmt.Sprintf("http://%s:%s@%s:%s", "admin", "pA_sw0rd", databaseHost, databasePort)
	adapter, err := adapters.GetCouchdbConnection(ctx, url)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(ctx context.Context, databaseName string, databaseUsername string, databasePassword string) error {
		url := fmt.Sprintf("http://%s:%s@%s:%s", databaseUsername, databasePassword, databaseHost, databasePort)
		client, err := kivik.New("couch", url)
		if err != nil {
			return err
		}
		defer client.Close(ctx)

		_, _, err = client.DB(databaseName).CreateDoc(ctx, map[string]interface{}{"test": "test"})
		return err
	}

	testHelper(t, ctx, adapter, clientConnectTest)
}
