# k8s-external-database-operator

This operator helps you to manage (self-)hosted databases in your kubernetes clusters by defining them with [Custom-Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).
It will automatically handle creation, updates & removal of your external databases.

> __Example:__ You are deploying a guestbook web-application. For the app you need some kind of deployment, a service, an ingress route and a database.
As you do already have an external database system hosted outside kubernetes you only want an automatic generation of a new database in that system with a new user for your application.

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

## Supported database types
- mysql
- couchdb
- mongo
- postgres

## Installation
### Requirements

## 
