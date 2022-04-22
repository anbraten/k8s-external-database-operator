package adapters_test

import (
	"context"
	"fmt"
	"testing"

	"database/sql"

	"github.com/anbraten/k8s-external-database-operator/adapters"
)

func TestPostgresDB(t *testing.T) {
	databaseHost := "localhost"
	databasePort := "5432"

	ctx := context.Background()
	url := fmt.Sprintf("postgres://%s:%s@%s:%s?sslmode=disable", "postgres", "pA_sw0rd", databaseHost, databasePort)
	adapter, err := adapters.GetPostgresConnection(ctx, url)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(ctx context.Context, databaseName string, databaseUsername string, databasePassword string) error {
		url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", databaseUsername, databasePassword, databaseHost, databasePort, databaseName)
		client, err := sql.Open("postgres", url)
		if err != nil {
			return err
		}

		_, err = client.ExecContext(ctx, "CREATE TABLE test (id int);")
		return err
	}

	testHelper(t, ctx, adapter, clientConnectTest)
}
