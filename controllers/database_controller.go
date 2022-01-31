package controllers

import (
	"context"
	"os"

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
		log.Error(err, "Failed to get Database")
		return ctrl.Result{}, err
	}

	db, err := r.getDatabaseConnection(ctx, database.Spec.Type)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer db.Close(ctx)

	log.Info("Connected to database server")

	hasDatabase, err := db.HasDatabase(ctx, database.Spec.Database)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !hasDatabase {
		log.Info("Create new database: '" + database.Spec.Database + "'")

		err = db.CreateDatabase(ctx, database.Spec.Database)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	hasDatabaseUserWithAccess, err := db.HasDatabaseUserWithAccess(ctx, database.Spec.Username, database.Spec.Database)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !hasDatabaseUserWithAccess {
		log.Info("Create new user '" + database.Spec.Username + "' with access to the database '" + database.Spec.Database + "'")

		err = db.CreateDatabaseUser(ctx, database.Spec.Username, database.Spec.Password, database.Spec.Database)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("Created database and user with full access to it")

	// Check if the Database instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isDatabaseMarkedToBeDeleted := database.GetDeletionTimestamp() != nil
	if isDatabaseMarkedToBeDeleted {
		if contains(database.GetFinalizers(), databaseFinalizer) {
			// Run finalization logic for databaseFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeDatabase(ctx, log, database); err != nil {
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

	// Add finalizer for this CR
	if !contains(database.GetFinalizers(), databaseFinalizer) {
		if err := r.addFinalizer(log, database); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *DatabaseReconciler) getDatabaseConnection(ctx context.Context, databaseType string) (adapters.DatabaseAdapter, error) {
	if databaseType == "mysql" {
		mysqlHost := os.Getenv("MYSQL_HOST")
		mysqlAdminUsername := os.Getenv("MYSQL_ADMIN_USERNAME")
		mysqlAdminPassword := os.Getenv("MYSQL_ADMIN_PASSWORD")

		if mysqlHost == "" || mysqlAdminUsername == "" || mysqlAdminPassword == "" {
			return nil, errors.NewBadRequest("Mysql database not configured (provide: MYSQL_HOST, MYSQL_ADMIN_USERNAME, MYSQL_ADMIN_PASSWORD)")
		}

		return adapters.GetMysqlConnection(ctx, mysqlHost, mysqlAdminUsername, mysqlAdminPassword)
	}

	if databaseType == "couchdb" {
		couchdbURL := os.Getenv("COUCHDB_URL")

		if couchdbURL == "" {
			return nil, errors.NewBadRequest("Couchdb database not configured (provide: COUCHDB_URL)")
		}

		return adapters.GetCouchdbConnection(ctx, couchdbURL)
	}

	if databaseType == "mongo" {
		mongoURL := os.Getenv("MONGO_URL")

		if mongoURL == "" {
			return nil, errors.NewBadRequest("Mongo database not configured (provide: MONGO_URL)")
		}

		return adapters.GetMongoConnection(ctx, mongoURL)
	}

	return nil, errors.NewBadRequest("Database type not supported")
}

func (r *DatabaseReconciler) finalizeDatabase(ctx context.Context, log logr.Logger, database *anbratengithubiov1alpha1.Database) error {
	db, err := r.getDatabaseConnection(ctx, database.Spec.Type)
	if err != nil {
		return err
	}

	defer db.Close(ctx)

	hasDatabaseUserWithAccess, err := db.HasDatabaseUserWithAccess(ctx, database.Spec.Username, database.Spec.Database)
	if err != nil {
		return err
	}

	if hasDatabaseUserWithAccess {
		log.Info("Remove user '" + database.Spec.Username + "' and its access to the database '" + database.Spec.Database + "'")

		err = db.DeleteDatabaseUser(ctx, database.Spec.Username, database.Spec.Database)
		if err != nil {
			return err
		}
	}

	hasDatabase, err := db.HasDatabase(ctx, database.Spec.Database)
	if err != nil {
		return err
	}

	if hasDatabase {
		log.Info("Remove database: '" + database.Spec.Database + "'")

		err = db.DeleteDatabase(ctx, database.Spec.Database)
		if err != nil {
			return err
		}
	}

	err = db.DeleteDatabase(ctx, database.Spec.Database)
	if err != nil {
		return err
	}

	log.Info("Database: '" + database.Spec.Database + "' and user: '" + database.Spec.Username + "' removed")
	return nil
}

func (r *DatabaseReconciler) addFinalizer(log logr.Logger, m *anbratengithubiov1alpha1.Database) error {
	log.Info("Adding Finalizer for the database")
	controllerutil.AddFinalizer(m, databaseFinalizer)

	// Update CR
	err := r.Update(context.TODO(), m)
	if err != nil {
		log.Error(err, "Failed to update database with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&anbratengithubiov1alpha1.Database{}).
		Complete(r)
}
