# External database operator for Kubernetes

This operator helps you to manage (self-)hosted databases in your Kubernetes clusters by defining them with [Custom-Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).
It will automatically handle creation, updates & removal of your external databases.

> __Example:__ You are deploying a guestbook web-application. For the app you need some kind of deployment, a service, an ingress route and a database.
As you do already have an external database system hosted outside Kubernetes you only want an automatic generation of a new database in that system with a new user for your application.

This can be done with the following manifest:

```yaml
apiVersion: rzab.de/v1
kind: Database
metadata:
  name: guestbook-database
spec:
  type: mysql
  database: guestbook
  username: guestbook-admin
  password: iwonttellyou
```

> Important note: :rotating_light: Database types can not be updated! Updating the user credentials normally results in recreation of the complete user which could delete your custom changes.

## Supported database types
- mysql
- couchdb
- mongo
- postgres

## Installation
### Requirements

##
