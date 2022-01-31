module github.com/anbraten/k8s-external-database-operator

go 1.16

require (
	github.com/go-kivik/couchdb/v4 v4.0.0-20220102153537-559e1c765356
	github.com/go-kivik/kivik/v4 v4.0.0-20220109201934-5b2b9d50be30
	github.com/go-logr/logr v0.3.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/snappy v0.0.3 // indirect
	github.com/onsi/ginkgo v1.16.1
	github.com/onsi/gomega v1.11.0
	github.com/xdg/scram v1.0.3 // indirect
	github.com/xdg/stringprep v1.0.3 // indirect
	go.mongodb.org/mongo-driver v1.1.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.7.2
)
