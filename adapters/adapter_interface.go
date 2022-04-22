package adapters

import "context"

type DatabaseAdapter interface {
	HasDatabase(ctx context.Context, database string) (bool, error)
	CreateDatabase(ctx context.Context, database string) error
	DeleteDatabase(ctx context.Context, database string) error
	HasDatabaseUserWithAccess(ctx context.Context, database string, username string) (bool, error)
	CreateDatabaseUser(ctx context.Context, database string, username string, quotedPassword string) error
	DeleteDatabaseUser(ctx context.Context, database string, username string) error
	Close(ctx context.Context) error
}
