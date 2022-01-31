package adapters

import "context"

type DatabaseAdapter interface {
	HasDatabase(ctx context.Context, database string) (bool, error)
	CreateDatabase(ctx context.Context, database string) error
	DeleteDatabase(ctx context.Context, database string) error
	HasDatabaseUserWithAccess(ctx context.Context, username string, database string) (bool, error)
	CreateDatabaseUser(ctx context.Context, username string, password string, database string) error
	DeleteDatabaseUser(ctx context.Context, username string, database string) error
	Close(ctx context.Context) error
}
