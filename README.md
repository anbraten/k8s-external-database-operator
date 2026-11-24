# External database operator for Kubernetes

This operator helps you to manage (self-)hosted databases in your Kubernetes clusters by defining them with [Custom-Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).
It will automatically handle creation, updates & removal of your external databases.

> **Example:** You are deploying a guestbook web-application. For the app you need some kind of deployment, a service, an ingress route and a database.
> As you do already have an external database system hosted outside Kubernetes you only want an automatic generation of a new database in that system with a new user for your application.

This can be done with the following manifest:

```yaml
apiVersion: anbraten.github.io/v1alpha1
kind: Database
metadata:
  name: guestbook-database
spec:
  type: mongo
  database: guestbook
  username: guestbook-admin
  password: iwonttellyou
```

> Important note: :rotating_light: Changing database settings (`type`, `database`, `username`, `password`) will possibly re-create the corresponding data and wont migrate the database or user (data-loss of old database & custom user settings).

## Supported database types

- mongo :white_check_mark:
- couchdb :white_check_mark:
- mysql :white_check_mark:
- postgres :white_check_mark:
- mssql :white_check_mark:

## Database type specific settings

### Postgres

Extensions (e.g. [pgvector](https://github.com/pgvector/pgvector)) can be created in the database by the operator.
As most extensions are not trusted, the database user can't create them on its own.
The extensions have to be installed on the postgres server (e.g. by using the `pgvector/pgvector` image).

```yaml
apiVersion: anbraten.github.io/v1alpha1
kind: Database
metadata:
  name: guestbook-database
spec:
  type: postgres
  database: guestbook
  username: guestbook-admin
  password: iwonttellyou
  postgres:
    extensions:
      - vector
```

Extensions are created on the next reconcile, so they can be added to existing databases as well.
Extensions removed from the list are **not** dropped, as this would delete all data using them (e.g. all `vector` columns).
The `postgres` settings can only be used with `type: postgres`.

## Deployment

1. Adjust the secrets in `deploy/database-secrets.yml` to your needs.
1. Deploy them using: `kubectl apply -f deploy/database-secrets.yml`
1. Deploy the controller using: `kubectl apply -f https://github.com/anbraten/k8s-external-database-operator/releases/latest/download/external-database-controller.yml`

## Upgrading

`kubectl apply` does not delete objects that have been removed from the manifest. After upgrading, delete them manually.

### From 0.0.x (kube-rbac-proxy) to the built-in metrics authentication

The `kube-rbac-proxy` sidecar has been replaced by the built-in authentication of the metrics endpoint (see [Metrics](#metrics)). The metrics service still listens on port `8443` (`https`), so existing scrapers keep working. Remove the leftover objects with:

```bash
kubectl delete --ignore-not-found clusterrolebinding external-database-operator-proxy-rolebinding
kubectl delete --ignore-not-found clusterrole external-database-operator-proxy-role
kubectl delete --ignore-not-found -n external-database-operator-system configmap external-database-operator-manager-config
```

PostgreSQL users created by older versions on PostgreSQL >= 15 are missing the privileges on the `public` schema. They are repaired automatically on the next reconcile after the upgrade.

## Metrics

The controller exposes Prometheus metrics via HTTPS on port `8443` (service `external-database-operator-controller-manager-metrics-service`).
Every request is authenticated and authorized against the Kubernetes API, so the scraping client has to send a service account token and needs permission to `get` the `/metrics` non-resource URL.
To grant this permission, bind the `external-database-operator-metrics-reader` ClusterRole to the service account of your scraper (e.g. Prometheus):

```bash
kubectl create clusterrolebinding prometheus-external-database-metrics \
  --clusterrole=external-database-operator-metrics-reader \
  --serviceaccount=<prometheus-namespace>:<prometheus-service-account>
```

## Development

Start the databases and run the adapter tests:

```bash
docker compose up -d --wait
go test ./adapters/...
```

Run all tests (including the controller tests using [envtest](https://book.kubebuilder.io/reference/envtest)):

```bash
make test
```

## Release

To release a new version of the controller run:

```bash
export VERSION="0.0.1"
export IMG="anbraten/external-database-operator:${VERSION}"
make docker-build
make docker-push
make generate-manifests
cat deploy/external-database-controller.yml
```
