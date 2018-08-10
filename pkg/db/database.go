package db

import (
	"github.com/golang/glog"
)

// DatabaseServer is the interface that wraps all db server implementations
//
// Create creates a database with the given name and provides access to the
// given User with the password. If the given User does not exits, it will
// create it. It returns an error if the creation fails.
// Create is immutable, so it will not return an error, if a database with
// that name already exists. But it will not grant access to the user. This
// prevents anyone with database resource creation privilege to access any
// preexisting database
//
// Delete delete a database with the given name. It returns an error if the
// deletion fails
// Delete is immutable, it will not return an error if no database with that
// name exists.
//
// Ping checks the connection and the credentials
type DatabaseServer interface {
	Create(dbName, user, password string) error
	Delete(dbName string) error
	Ping(dbName, user, password string) bool
}

type MockDbServer struct{}

func (MockDbServer) Create(dbName, user, password string) error {
	glog.Infoln("Deploying Mock Database")
	return nil
}

func (MockDbServer) Delete(dnName string) error {
	glog.Infoln("Deleting Mock Database")
	return nil
}


func (MockDbServer) Ping(dbName, user, password string) bool {
	return user != ""
}
