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
	couchdbUrl := fmt.Sprintf("http://admin:1234@%s:%s", databaseHost, databasePort)
	adapter, err := adapters.GetCouchdbConnection(ctx, couchdbUrl)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(databaseName string, databaseUsername string, databasePassword string) error {
		client, err := kivik.New("couch", fmt.Sprintf("http://%s:%s@%s:%s", databaseUsername, databasePassword, databaseHost, databasePort))
		if err != nil {
			return err
		}
		_, _, err = client.DB(databaseName).CreateDoc(ctx, map[string]interface{}{"test": "test"})
		return err
	}

	testHelper(t, ctx, adapter, clientConnectTest)
}
