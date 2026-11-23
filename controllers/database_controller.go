package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	adapters "github.com/anbraten/k8s-external-database-operator/adapters"
	anbratengithubiov1alpha1 "github.com/anbraten/k8s-external-database-operator/api/v1alpha1"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	// APIReader reads directly from the API server. It is used to read secrets, so
	// only their metadata (for the watch) and not their data is cached.
	APIReader client.Reader
	Log       logr.Logger
	Scheme    *runtime.Scheme
}

const databaseFinalizer = "finalizer.database.anbraten.github.io"

// passwordSecretRefIndex indexes databases by the name of the secret referenced by passwordSecretRef
const passwordSecretRefIndex = ".spec.passwordSecretRef.name"

//+kubebuilder:rbac:groups=anbraten.github.io,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=anbraten.github.io,resources=databases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=anbraten.github.io,resources=databases/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
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

	// check if database and username is sql safe if not mongo and not couchdb
	if database.Spec.Type != "mongo" && database.Spec.Type != "couchdb" {
		if err = adapters.IsValidIdentifier(database.Spec.Database); err != nil {
			log.Error(err, fmt.Sprintf("Please make sure your database name matches '%s'", adapters.IdentifierRegex.String()))
			return ctrl.Result{}, err
		}

		if err = adapters.IsValidIdentifier(database.Spec.Username); err != nil {
			log.Error(err, fmt.Sprintf("Please make sure your database username matches '%s'", adapters.IdentifierRegex.String()))
			return ctrl.Result{}, err
		}
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

	password, passwordVersion, err := r.getPassword(ctx, database)
	if err != nil {
		log.Error(err, "Can't load password of database user")
		return ctrl.Result{}, err
	}

	// Create database user with full access if necessary
	hasDatabaseUserWithAccess, err := db.HasDatabaseUserWithAccess(ctx, database.Spec.Database, database.Spec.Username)
	if err != nil {
		log.Error(err, "Can't check if user has access to database")
		return ctrl.Result{}, err
	} else if !hasDatabaseUserWithAccess {
		if password == "" {
			err = fmt.Errorf("neither password nor passwordSecretRef is set")
			log.Error(err, "Can't create database user without a password")
			return ctrl.Result{}, err
		}

		log.Info("Creating new user and granting access to database")

		err = db.CreateDatabaseUser(ctx, database.Spec.Database, database.Spec.Username, password)
		if err != nil {
			log.Error(err, "Can't create database user with access to database")
			return ctrl.Result{}, err
		}
		log.Info("Created user and granted access to database")
	}

	// Update the password of the user if it has been changed. CreateDatabaseUser keeps the password
	// of an already existing user for some adapters, so it is also applied after creating the user.
	// Without any password (old resources) the password of the user is left untouched.
	if password != "" && database.Status.PasswordVersion != passwordVersion {
		log.Info("Setting password of database user")

		err = db.UpdateDatabaseUserPassword(ctx, database.Spec.Database, database.Spec.Username, password)
		if err != nil {
			log.Error(err, "Can't set password of database user")
			return ctrl.Result{}, err
		}

		patch := client.MergeFrom(database.DeepCopy())
		// created is required by the status schema, and database and user exist at this point
		database.Status.Created = true
		database.Status.PasswordVersion = passwordVersion
		if err = r.Status().Patch(ctx, database, patch); err != nil {
			log.Error(err, "Can't update status of database")
			return ctrl.Result{}, err
		}

		log.Info("Set password of database user")
	}

	return ctrl.Result{}, nil
}

// getPassword returns the password of the database user, either set directly or loaded from the
// referenced secret in the namespace of the database, and a version which changes whenever the
// password (or its source) may have changed. Without any password an empty password is returned.
func (r *DatabaseReconciler) getPassword(ctx context.Context, database *anbratengithubiov1alpha1.Database) (string, string, error) {
	ref := database.Spec.PasswordSecretRef

	// also rejected by the CRD validation, but resources created before it was added may still have both
	if ref != nil && database.Spec.Password != "" {
		return "", "", fmt.Errorf("only one of password or passwordSecretRef may be set")
	}

	if ref == nil {
		if database.Spec.Password == "" {
			return "", "", nil
		}
		// the generation changes with every change of the spec (including the password)
		return database.Spec.Password, fmt.Sprintf("spec/%d", database.Generation), nil
	}

	secret := &corev1.Secret{}
	if err := r.APIReader.Get(ctx, types.NamespacedName{Namespace: database.Namespace, Name: ref.Name}, secret); err != nil {
		return "", "", fmt.Errorf("failed to get secret '%s': %w", ref.Name, err)
	}

	password, ok := secret.Data[ref.Key]
	if !ok {
		return "", "", fmt.Errorf("secret '%s' has no key '%s'", ref.Name, ref.Key)
	}
	if len(password) == 0 {
		return "", "", fmt.Errorf("key '%s' of secret '%s' is empty", ref.Key, ref.Name)
	}

	// the uid changes if the secret is re-created, the resource version with every change of it
	return string(password), fmt.Sprintf("secret/%s/%s/%s/%s", ref.Name, ref.Key, secret.UID, secret.ResourceVersion), nil
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
	err := mgr.GetFieldIndexer().IndexField(context.Background(), &anbratengithubiov1alpha1.Database{}, passwordSecretRefIndex, indexPasswordSecretRef)
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&anbratengithubiov1alpha1.Database{}).
		// only the metadata of secrets is watched and cached, the data is read on demand using the APIReader
		Watches(&corev1.Secret{}, handler.EnqueueRequestsFromMapFunc(r.databasesForSecret), builder.OnlyMetadata).
		Complete(r)
}

func indexPasswordSecretRef(obj client.Object) []string {
	ref := obj.(*anbratengithubiov1alpha1.Database).Spec.PasswordSecretRef
	if ref == nil {
		return nil
	}
	return []string{ref.Name}
}

// databasesForSecret returns reconcile requests for all databases referencing the secret
func (r *DatabaseReconciler) databasesForSecret(ctx context.Context, secret client.Object) []reconcile.Request {
	databases := &anbratengithubiov1alpha1.DatabaseList{}
	err := r.List(ctx, databases, client.InNamespace(secret.GetNamespace()), client.MatchingFields{passwordSecretRefIndex: secret.GetName()})
	if err != nil {
		r.Log.Error(err, "Failed to list databases referencing secret", "secret", client.ObjectKeyFromObject(secret))
		return nil
	}

	requests := make([]reconcile.Request, 0, len(databases.Items))
	for _, database := range databases.Items {
		requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&database)})
	}
	return requests
}
