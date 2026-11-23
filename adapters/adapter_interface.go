package adapters

import (
	"context"

	"github.com/anbraten/k8s-external-database-operator/api/v1alpha1"
)

type DatabaseAdapter interface {
	HasDatabase(ctx context.Context, database string) (bool, error)
	CreateDatabase(ctx context.Context, database string) error
	DeleteDatabase(ctx context.Context, database string) error
	HasDatabaseUserWithAccess(ctx context.Context, database string, username string) (bool, error)
	CreateDatabaseUser(ctx context.Context, database string, username string, password string) error
	DeleteDatabaseUser(ctx context.Context, database string, username string) error
	Close(ctx context.Context) error
}

// SpecificSpecAdapter is implemented by adapters supporting settings which are specific to their
// database type (e.g. spec.postgres). ApplyAdapterSpecificSpec is called on every reconcile after
// the database and user have been created, so it has to be idempotent.
type SpecificSpecAdapter interface {
	ApplyAdapterSpecificSpec(ctx context.Context, spec v1alpha1.DatabaseSpec) error
}
