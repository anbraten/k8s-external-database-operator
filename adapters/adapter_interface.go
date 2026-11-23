package adapters

import "context"

type DatabaseAdapter interface {
	HasDatabase(ctx context.Context, database string) (bool, error)
	CreateDatabase(ctx context.Context, database string) error
	DeleteDatabase(ctx context.Context, database string) error
	HasDatabaseUserWithAccess(ctx context.Context, database string, username string) (bool, error)
	CreateDatabaseUser(ctx context.Context, database string, username string, password string) error
	DeleteDatabaseUser(ctx context.Context, database string, username string) error
	Close(ctx context.Context) error
}

// ExtensionAdapter is implemented by adapters of databases supporting extensions (postgres).
type ExtensionAdapter interface {
	// EnsureExtensions creates all given extensions in the database which do not exist yet.
	// Existing extensions which are not part of the list are kept.
	EnsureExtensions(ctx context.Context, database string, extensions []string) error
}
