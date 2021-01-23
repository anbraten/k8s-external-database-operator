package adapters

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	// TODO check is this a proper import?
)

type mysqlAdapter struct {
	host          string
	adminUsername string
	adminPassword string
	db            *sql.DB
}

func (adapter mysqlAdapter) runSQL(sql string) error {
	insert, err := adapter.db.Query(sql)

	// if there is an error inserting, handle it
	if err != nil {
		return err
	}

	// be careful deferring Queries if you are using transactions
	defer insert.Close()

	return nil
}

func (adapter mysqlAdapter) CreateDatabase(name string) error {
	// TODO use proper sql query
	return adapter.runSQL("CREATE db")
}

func (adapter mysqlAdapter) DeleteDatabase(name string) error {
	// TODO use proper sql query
	return adapter.runSQL("DROP db")
}

func (adapter mysqlAdapter) UpdateDatabaseUser(username string, password string) error {
	// TODO implement
	return nil
}

func (adapter mysqlAdapter) Close() error {
	return adapter.db.Close()
}

func createMysql(host string, adminUsername string, adminPassword string) (*mysqlAdapter, error) {
	db, err := sql.Open("mysql", adminUsername+":"+adminPassword+"@"+host)
	if err != nil {
		return nil, err
	}

	adapter := mysqlAdapter{
		host:          host,
		db:            db,
		adminUsername: adminUsername,
		adminPassword: adminPassword,
	}

	return &adapter, nil
}
