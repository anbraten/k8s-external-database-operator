package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatabaseSpec defines the desired state of Database
// +kubebuilder:validation:XValidation:rule="!has(self.postgres) || (has(self.type) && self.type == 'postgres')",message="postgres can only be set if type is postgres"
type DatabaseSpec struct {
	// Type of the database. (currently supported: mysql, couchdb, mongo, postgres, mssql)
	Type string `json:"type,omitempty"`

	// Name of the database
	Database string `json:"database,omitempty"`

	// Username of the database user which will have full access to this database
	Username string `json:"username,omitempty"`

	// Password of the database user
	Password string `json:"password,omitempty"`

	// Postgres specific settings (only allowed if type is postgres)
	// +optional
	Postgres *PostgresSpec `json:"postgres,omitempty"`
}

// PostgresSpec defines settings which are only supported by postgres databases
type PostgresSpec struct {
	// Extensions which will be created in the database (e.g. vector, postgis, pg_trgm).
	// The extensions have to be installed on the postgres server.
	// Extensions removed from this list are not dropped from the database, as this would delete all data using them.
	// +optional
	// +listType=set
	// +kubebuilder:validation:items:Pattern=`^[a-zA-Z0-9]+?[a-zA-Z0-9_-]*?$`
	Extensions []string `json:"extensions,omitempty"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// Has database been created?
	Created bool `json:"created"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Database is the Schema for the databases API
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatabaseList contains a list of Database
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}
