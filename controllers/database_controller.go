/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
			log.Info("Database resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Database")
		return ctrl.Result{}, err
	}

	db, err := r.getDatabaseConnection(database.Spec.Type)
	if err != nil {
		return ctrl.Result{}, err
	}

	err, hasDatabase := db.HasDatabase(database.Spec.Database)
	if !hasDatabase {
		log.Info("Create new database")

		err = db.CreateDatabase(database.Spec.Database)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	err, hasDatabaseUserWithAccess := db.HasDatabaseUserWithAccess(database.Spec.Username, database.Spec.Database)
	if !hasDatabaseUserWithAccess {
		log.Info("Create new user with access to the database")

		err = db.UpdateDatabaseUser(database.Spec.Username, database.Spec.Password, database.Spec.Database)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	err = db.Close()
	if err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Created new database '" + database.Spec.Database + "' with full access for user: '" + database.Spec.Username + "'")

	// Check if the Database instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isDatabaseMarkedToBeDeleted := database.GetDeletionTimestamp() != nil
	if isDatabaseMarkedToBeDeleted {
		if contains(database.GetFinalizers(), databaseFinalizer) {
			// Run finalization logic for databaseFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeDatabase(log, database); err != nil {
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

func (r *DatabaseReconciler) getDatabaseConnection(databaseType string) (adapters.DatabaseAdapter, error) {
	if databaseType == "mysql" {
		mysqlHost := os.Getenv("MYSQL_HOST")
		mysqlAdminUsername := os.Getenv("MYSQL_ADMIN_USERNAME")
		mysqlAdminPassword := os.Getenv("MYSQL_ADMIN_PASSWORD")
		return adapters.CreateConnection("mysql", mysqlHost, mysqlAdminUsername, mysqlAdminPassword)
	}

	if databaseType == "couchdb" {
		couchdbURL := os.Getenv("COUCHDB_URL")
		couchdbAdminUsername := os.Getenv("COUCHDB_ADMIN_USERNAME")
		couchdbAdminPassword := os.Getenv("COUCHDB_ADMIN_PASSWORD")
		return adapters.CreateConnection("couchdb", couchdbURL, couchdbAdminUsername, couchdbAdminPassword)
	}

	return nil, errors.NewBadRequest("Database type not supported")
}

func (r *DatabaseReconciler) finalizeDatabase(log logr.Logger, database *anbratengithubiov1alpha1.Database) error {
	db, err := r.getDatabaseConnection(database.Spec.Type)
	if err != nil {
		return err
	}

	err = db.DeleteDatabase(database.Spec.Database)
	if err != nil {
		return err
	}

	err = db.Close()
	if err != nil {
		return err
	}

	log.Info("Successfully removed database: '" + database.Spec.Database + "'")
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
