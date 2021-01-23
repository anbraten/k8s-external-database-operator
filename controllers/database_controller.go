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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/anbraten/k8s-external-database-operator/adapters"
	rzabdev1 "github.com/anbraten/k8s-external-database-operator/api/v1"
)

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const databaseFinalizer = "finalizer.database.rzab.de"

// +kubebuilder:rbac:groups=rzab.de,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rzab.de,resources=databases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rzab.de,resources=databases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("database", req.NamespacedName)

	// Fetch the Database instance
	database := &rzabdev1.Database{}
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

	log.Info("Create new database")

	// TODO check if database already exists
	// TODO create new database if not
	// TODO update if exits and updates are necessary

	db, err := r.getDatabaseConnection("mysql") // TODO use type from manifest
	if err != nil {
		return ctrl.Result{}, err
	}

	err = db.CreateDatabase("test") // TODO use db name from manifest
	if err != nil {
		return ctrl.Result{}, err
	}

	err = db.DeleteDatabase("test") // TODO use db name from manifest
	if err != nil {
		return ctrl.Result{}, err
	}

	err = db.Close()
	if err != nil {
		return ctrl.Result{}, err
	}

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
		mysqlHost := "localhost"       // TODO use env
		mysqlAdminUsername := "admin"  // TODO use env
		mysqlAdminPassword := "secure" // TODO use env
		return adapters.CreateConnection("mysql", mysqlHost, mysqlAdminUsername, mysqlAdminPassword)
	}

	if databaseType == "couchdb" {
		couchdbURL := "http://localhost" // TODO use env
		couchdbAdminUsername := "admin"  // TODO use env
		couchdbAdminPassword := "secure" // TODO use env
		return adapters.CreateConnection("couchdb", couchdbURL, couchdbAdminUsername, couchdbAdminPassword)
	}

	return adapters.CreateConnection(databaseType, "", "", "")
}

func (r *DatabaseReconciler) finalizeDatabase(log logr.Logger, m *rzabdev1.Database) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	log.Info("Successfully removed database")
	return nil
}

func (r *DatabaseReconciler) addFinalizer(log logr.Logger, m *rzabdev1.Database) error {
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
		For(&rzabdev1.Database{}).
		Complete(r)
}