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

> Important note: :rotating_light: Database settings (`type`, `database`, `username`, `password`) will possibly re-create the corresponding data and wont migrate the database or user.

## Supported database types

- mongo :white_check_mark:
- mysql :hammer:
- couchdb :hammer:
- postgres :clock1:

## Installation

### Requirements

### Deployment
