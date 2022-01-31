package adapters_test

import (
	"context"
	"testing"

	"github.com/anbraten/k8s-external-database-operator/adapters"
)

func TestMongoDB(t *testing.T) {
	ctx := context.Background()
	mongodbUrl := "mongodb://admin:1234@localhost:27017/?authSource=admin"
	adapter, err := adapters.GetMongoConnection(ctx, mongodbUrl)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	testHelper(t, ctx, adapter)
}
