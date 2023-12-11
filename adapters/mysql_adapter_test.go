package adapters_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
)

func prepareMySqlDB(t *testing.T) (context.Context, adapters.DatabaseAdapter, ClientConnectTest) {
	databaseHost := "localhost"
	databasePort := "3306"

	ctx := context.Background()
	url := fmt.Sprintf("%s:%s@tcp(%s:%s)/", "root", "pA_sw0rd", databaseHost, databasePort)
	adapter, err := adapters.GetMysqlConnection(ctx, url)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(ctx context.Context, databaseName string, databaseUsername string, databasePassword string) error {
		url := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", databaseUsername, databasePassword, databaseHost, databasePort, databaseName)
		client, err := sql.Open("mysql", url)
		if err != nil {
			return err
		}
		defer client.Close()

		_, err = client.ExecContext(ctx, "CREATE TABLE test (id int);")
		return err
	}

	return ctx, adapter, clientConnectTest
}

func TestMySqlDB(t *testing.T) {
	ctx, adapter, clientConnectTest := prepareMySqlDB(t)

	testHelper(t, ctx, adapter, clientConnectTest)
}

func TestMySqlDBCleanup(t *testing.T) {
	ctx, adapter, _ := prepareMySqlDB(t)

	cleanupTestHelper(t, ctx, adapter)
}
