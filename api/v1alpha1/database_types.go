package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SecretKeyReference references a key of a secret in the namespace of the Database
type SecretKeyReference struct {
	// Name of the secret
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Key in the secret which contains the value
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`
}

// DatabaseSpec defines the desired state of Database
// +kubebuilder:validation:XValidation:rule="!(has(self.password) && has(self.passwordSecretRef))",message="only one of password or passwordSecretRef may be set"
type DatabaseSpec struct {
	// Type of the database. (currently supported: mysql, couchdb, mongo, postgres)
	Type string `json:"type,omitempty"`

	// Name of the database
	Database string `json:"database,omitempty"`

	// Username of the database user which will have full access to this database
	Username string `json:"username,omitempty"`

	// Password of the database user. Prefer passwordSecretRef to keep the password out of the resource.
	Password string `json:"password,omitempty"`

	// Reference to a key of a secret in the namespace of the Database which contains the password of the database user
	PasswordSecretRef *SecretKeyReference `json:"passwordSecretRef,omitempty"`
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
