package adapters_test

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
)

func prepareMsSqlDB(t *testing.T) (context.Context, adapters.DatabaseAdapter, ClientConnectTest) {
	databaseHost := "localhost"
	databasePort := "1433"

	ctx := context.Background()
	adminURL := fmt.Sprintf("sqlserver://%s:%s@%s:%s", "sa", "pA_sw0rd", databaseHost, databasePort)
	adapter, err := adapters.GetMssqlConnection(ctx, adminURL)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(ctx context.Context, databaseName string, databaseUsername string, databasePassword string) error {
		dsn := url.URL{
			Scheme:   "sqlserver",
			User:     url.UserPassword(databaseUsername, databasePassword),
			Host:     net.JoinHostPort(databaseHost, databasePort),
			RawQuery: url.Values{"database": {databaseName}}.Encode(),
		}
		client, err := sql.Open("sqlserver", dsn.String())
		if err != nil {
			return err
		}
		defer client.Close()

		_, err = client.ExecContext(ctx, "IF OBJECT_ID('test') IS NULL CREATE TABLE test (id int);")
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
