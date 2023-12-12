package adapters_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
)

func prepareMsSqlDB(t *testing.T) (context.Context, adapters.DatabaseAdapter, ClientConnectTest) {
	databaseHost := "localhost"
	databasePort := "1433"

	ctx := context.Background()
	url := fmt.Sprintf("sqlserver://%s:%s@%s:%s", "sa", "pA_sw0rd", databaseHost, databasePort)
	adapter, err := adapters.GetMssqlConnection(ctx, url)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(ctx context.Context, databaseName string, databaseUsername string, databasePassword string) error {
		url := fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=%s", databaseUsername, databasePassword, databaseHost, databasePort, databaseName)
		client, err := sql.Open("sqlserver", url)
		if err != nil {
			return err
		}
		defer client.Close()

		_, err = client.ExecContext(ctx, "CREATE TABLE test (id int);")
		return err
	}

	return ctx, adapter, clientConnectTest
}

func TestMsSqlDB(t *testing.T) {
	ctx, adapter, clientConnectTest := prepareMsSqlDB(t)

	testHelper(t, ctx, adapter, clientConnectTest)
}

func TestMsSqlDBCleanup(t *testing.T) {
	ctx, adapter, _ := prepareMsSqlDB(t)

	cleanupTestHelper(t, ctx, adapter)
}
