package adapters_test

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
	"github.com/go-sql-driver/mysql"
)

func prepareMySqlDB(t *testing.T) (context.Context, adapters.DatabaseAdapter, ClientConnectTest) {
	databaseHost := "localhost"
	databasePort := "3306"

	ctx := context.Background()
	adminURL := fmt.Sprintf("%s:%s@tcp(%s:%s)/", "root", "pA_sw0rd", databaseHost, databasePort)
	adapter, err := adapters.GetMysqlConnection(ctx, adminURL)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	clientConnectTest := func(ctx context.Context, databaseName string, databaseUsername string, databasePassword string) error {
		config := mysql.NewConfig()
		config.User = databaseUsername
		config.Passwd = databasePassword
		config.Net = "tcp"
		config.Addr = net.JoinHostPort(databaseHost, databasePort)
		config.DBName = databaseName
		client, err := sql.Open("mysql", config.FormatDSN())
		if err != nil {
			return err
		}
		defer client.Close()

		_, err = client.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS test (id int);")
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
