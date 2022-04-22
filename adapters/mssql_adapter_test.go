package adapters_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
)

func TestMsSqlDB(t *testing.T) {
	databaseHost := "localhost"
	databasePort := "1433"

	ctx := context.Background()
	url := fmt.Sprintf("mssql://%s:%s@%s:%s", "sa", "pA%sw0rd", databaseHost, databasePort)
	adapter, err := adapters.GetCouchdbConnection(ctx, url)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(ctx context.Context, databaseName string, databaseUsername string, databasePassword string) error {
		url := fmt.Sprintf("sqlserver://%s:%s@%s:%s", databaseUsername, databasePassword, databaseHost, databasePort)
		client, err := sql.Open("sqlserver", url)
		if err != nil {
			return err
		}

		_, err = client.ExecContext(ctx, "CREATE TABLE test (id int);")
		return err
	}

	testHelper(t, ctx, adapter, clientConnectTest)
}
