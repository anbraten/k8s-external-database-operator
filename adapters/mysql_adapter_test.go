package adapters_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
)

func TestMySqlDB(t *testing.T) {
	databaseHost := "localhost"
	databasePort := "3306"

	ctx := context.Background()
	url := fmt.Sprintf("%s:%s@tcp(%s:%s)/", "root", "pA%sw0rd", databaseHost, databasePort)
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

		_, err = client.ExecContext(ctx, "CREATE TABLE test (id int);")
		return err
	}

	testHelper(t, ctx, adapter, clientConnectTest)
}
