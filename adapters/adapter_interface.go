package adapters

type DatabaseAdapter interface {
	HasDatabase(database string) (bool, error)
	CreateDatabase(database string) error
	DeleteDatabase(database string) error
	HasDatabaseUserWithAccess(username string, database string) (bool, error)
	CreateDatabaseUser(username string, password string, database string) error
	DeleteDatabaseUser(username string, database string) error
	Close() error
}
