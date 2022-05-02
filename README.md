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

## Deployment

1. Adjust the secrets in `deploy/database-secrets.yml` to your needs.
1. Deploy them using: `kubectl apply -f deploy/database-secrets.yml`
1. Deploy the controller using: `kubectl apply -f https://github.com/anbraten/k8s-external-database-operator/releases/latest/download/external-database-controller.yml`

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
