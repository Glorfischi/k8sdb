# k8sDB

State: v0.0.1-alpha - use with caution

A Custom Resource Definition for provisioning databases on an existing database server. It allows users to provision databases
for their application. They can do this by creating a "Database" resource in Kuberenetes, the controller will then create a 
Database on the configured Database Server.

This is useful for CI builds and any system with frequent database deployments. Use this controller with caution, there are some precautions to prevent data loss in production environments, but we do not give any guarantees.

## Setup
tdb

## Usage 
Assuming you are using the sample configuration, to provision a database create a Database resource and Secret similar to this

    ---
    apiVersion: k8sdb.fischi.me/v1alpha1
    kind: Database
    metadata:
      name: foo-bar
    spec:
      type: postgres-staging
      credentials: foo-secret
      
The name of the resource, in this case `foo-bar` will also be the name of the actual database.

`type` is the type of database configured earlier.  In our case this is a staging postgres database on localhost.

`credentials` references the secret containing the username and password for the user that is used to access the database.

    ---
    apiVersion: v1
    kind: Secret
    metadata:
      name: foo-secret
      labels:
        database: foo
    data:
      password: MTIzNA==
      user: YWRtaW4=
    type: Opaque

The secret contains two values.

`user` which is the username of the db user
`password` which is the password of the db user

Assuming there exist no database called `foo-bar` on the database server, applying these two resources will provision a database called `foo-bar` and create a user called `admin`  with password `1234` (The values in Secrets are base64 encoded) that has all access granted on `foo-bar`.

Every Database resource that uses the user secret will add a finalizer to the secret. This means it should not be possible to delete a secret as long as a database is still using this user.

If there already exists a database called `foo-bar` on the database server, applying these resources will not provision a db and will not create a user.

Deleting the Database resource will then also drop the corresponding database. Users will not be deleted when deleting the corresponding secret.
