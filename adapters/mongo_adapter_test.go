package adapters_test

func TestMongoDB(t *testing.T) {
	ctx := context.Background()
	mongodbUrl := "mongodb://admin:1234@localhost:27018/?authSource=admin"
	adapter, err := adapters.GetMongoDb(ctx, mongodbUrl)
	if err != nil {
		t.Fatalf("Error opening database connection: %s", err)
	}

	testHelper(t, ctx, adapter)
}
