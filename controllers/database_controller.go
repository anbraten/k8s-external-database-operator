package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	adapters "github.com/anbraten/k8s-external-database-operator/adapters"
	anbratengithubiov1alpha1 "github.com/anbraten/k8s-external-database-operator/api/v1alpha1"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const databaseFinalizer = "finalizer.database.anbraten.github.io"

//+kubebuilder:rbac:groups=anbraten.github.io,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=anbraten.github.io,resources=databases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=anbraten.github.io,resources=databases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("database", req.NamespacedName)
	log.Info("Reconciling database")

	// Fetch the Database instance
	database := &anbratengithubiov1alpha1.Database{}
	err := r.Get(ctx, req.NamespacedName, database)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			// log.Info("Database resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get database resource")
		return ctrl.Result{}, err
	}

	// add loaded database details
	log = log.WithValues("database", database.Spec.Database, "username", database.Spec.Username)

	if err = adapters.IsValidIdentifier(database.Spec.Database); err != nil {
		log.Error(err, fmt.Sprintf("Please make sure your database name matches '%s'", adapters.IdentifierRegex.String()))
		return ctrl.Result{}, err
	}

	if err = adapters.IsValidIdentifier(database.Spec.Username); err != nil {
		log.Error(err, fmt.Sprintf("Please make sure your database username matches '%s'", adapters.IdentifierRegex.String()))
		return ctrl.Result{}, err
	}

	// use maximum of 3 seconds to connect
	openCtx, cancel := context.WithTimeout(ctx, time.Duration(time.Second*3))
	defer cancel()

	db, err := r.getDatabaseConnection(openCtx, database.Spec.Type)
	if err != nil {
		log.Error(err, "Failed to detect or open database connection")
		return ctrl.Result{}, err
	}

	defer db.Close(ctx)

	// Check if the Database instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isDatabaseMarkedToBeDeleted := database.GetDeletionTimestamp() != nil
	if isDatabaseMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(database, databaseFinalizer) {
			// Run finalization logic for databaseFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeDatabase(ctx, log, db, database); err != nil {
				log.Error(err, "Can't remove database and user")
				return ctrl.Result{}, err
			}

			// Remove databaseFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(database, databaseFinalizer)
			err := r.Update(ctx, database)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR if necessary
	if !controllerutil.ContainsFinalizer(database, databaseFinalizer) {
		controllerutil.AddFinalizer(database, databaseFinalizer)
		err = r.Update(ctx, database)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create database if necessary
	hasDatabase, err := db.HasDatabase(ctx, database.Spec.Database)
	if err != nil {
		log.Error(err, "Couldn't check if database exists")
		return ctrl.Result{}, err
	} else if !hasDatabase {
		log.Info("Creating new database '" + database.Spec.Database + "'")

		err = db.CreateDatabase(ctx, database.Spec.Database)
		if err != nil {
			log.Error(err, "Can't create database")
			return ctrl.Result{}, err
		}

		log.Info("Created database '" + database.Spec.Database + "'")
	}

	// Create database user with full access if necessary
	hasDatabaseUserWithAccess, err := db.HasDatabaseUserWithAccess(ctx, database.Spec.Database, database.Spec.Username)
	if err != nil {
		log.Error(err, "Can't check if user has access to database")
		return ctrl.Result{}, err
	} else if !hasDatabaseUserWithAccess {
		log.Info("Creating new user and granting access to database")

		err = db.CreateDatabaseUser(ctx, database.Spec.Database, database.Spec.Username, database.Spec.Password)
		if err != nil {
			log.Error(err, "Can't create database user with access to database")
			return ctrl.Result{}, err
		}
		log.Info("Created user and granted access to database")
	}

	return ctrl.Result{}, nil
}

func (r *DatabaseReconciler) getDatabaseConnection(ctx context.Context, databaseType string) (adapters.DatabaseAdapter, error) {
	switch databaseType {
	case "mysql":
		mysqlDSN := os.Getenv("MYSQL_DSN")
		if mysqlDSN == "" {
			return nil, errors.NewBadRequest("Mysql database not configured (provide: MYSQL_DSN)")
		}
		return adapters.GetMysqlConnection(ctx, mysqlDSN)

	case "couchdb":
		couchdbURL := os.Getenv("COUCHDB_URL")
		if couchdbURL == "" {
			return nil, errors.NewBadRequest("Couchdb database not configured (provide: COUCHDB_URL)")
		}
		return adapters.GetCouchdbConnection(ctx, couchdbURL)

	case "mongo":
		mongoURL := os.Getenv("MONGO_URL")
		if mongoURL == "" {
			return nil, errors.NewBadRequest("Mongo database not configured (provide: MONGO_URL)")
		}
		return adapters.GetMongoConnection(ctx, mongoURL)

	case "postgres":
		postgresURL := os.Getenv("POSTGRES_URL")
		if postgresURL == "" {
			return nil, errors.NewBadRequest("Postgres database not configured (provide: POSTGRES_URL)")
		}
		return adapters.GetPostgresConnection(ctx, postgresURL)

	case "mssql":
		mssqlURL := os.Getenv("MSSQL_URL")
		if mssqlURL == "" {
			return nil, errors.NewBadRequest("MS-SQL database not configured (provide: MSSQL_URL)")
		}
		return adapters.GetMssqlConnection(ctx, mssqlURL)

	default:
		return nil, errors.NewBadRequest("Database type not supported")
	}
}

func (r *DatabaseReconciler) finalizeDatabase(ctx context.Context, log logr.Logger, db adapters.DatabaseAdapter, database *anbratengithubiov1alpha1.Database) error {
	// remove database user and database access if it exists
	hasDatabaseUserWithAccess, err := db.HasDatabaseUserWithAccess(ctx, database.Spec.Database, database.Spec.Username)
	if err != nil {
		return err
	} else if hasDatabaseUserWithAccess {
		log.Info("Removing user and revoking access to the database")

		err = db.DeleteDatabaseUser(ctx, database.Spec.Database, database.Spec.Username)
		if err != nil {
			return err
		}
		log.Info("Removed database user and revoked access to the database")
	}

	// remove database if it exists
	hasDatabase, err := db.HasDatabase(ctx, database.Spec.Database)
	if err != nil {
		return err
	} else if hasDatabase {
		log.Info("Removing database")

		err = db.DeleteDatabase(ctx, database.Spec.Database)
		if err != nil {
			return err
		}
		log.Info("Removed database")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&anbratengithubiov1alpha1.Database{}).
		Complete(r)
}
