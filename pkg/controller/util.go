package controller

import (
	clientset "github.com/glorfischi/k8sdb/pkg/client/clientset/versioned"
	"github.com/glorfischi/k8sdb/pkg/apis/k8sdb/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/api/core/v1"
)

func updateDbObj(dbClient clientset.Interface, dbObj *v1alpha1.Database) error {
	_, err := dbClient.FischiV1alpha1().Databases(dbObj.Namespace).Update(dbObj)
	return err
}

func updateSecret(client *kubernetes.Clientset, secret *v1.Secret) error {
	_, err := client.CoreV1().Secrets(secret.Namespace).Update(secret)
	return err
}

type Finalizable interface {
	GetFinalizers() []string
	SetFinalizers([]string)
}

// A finalizer is logically connected to the actual database on the remote server. This means, when we create a
// database we also add a finalizer. When we delete the actual database, we also delete the finalizer.
// This has the effect, that  we cant't delete a resource with a database that still exists.
func hasFinalizer(obj Finalizable, finalizer string) bool {
	currentFinalizers := obj.GetFinalizers()
	for _, f := range currentFinalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}

// Adds a finalizer to the object if it is not added already. Note: Add finalizer changes the object passed to it
// So you will need to pass a pointer
func addFinalizer(obj Finalizable, finalizer string) {
	if hasFinalizer(obj, finalizer) {
		return
	}

	finalizers := obj.GetFinalizers()
	obj.SetFinalizers(append(finalizers, finalizer))
}

// Delete all finalizers on that object. Note: Add finalizer changes the object passed to it
// So you will need to pass a pointer
func deleteAllFinalizers(obj Finalizable) {
	obj.SetFinalizers([]string{})
}

// Delete  finalizers on that object. (There should always be only one) Deleting all finalizers also deletes the
// resource.
func deleteFinalizer(obj Finalizable, finalizer string) Finalizable {
	currentFinalizers := obj.GetFinalizers()
	newFinalizers := []string{}
	for _, f := range currentFinalizers {
		if f != finalizer {
			newFinalizers = append(newFinalizers, f)
		}
	}
	obj.SetFinalizers(newFinalizers)
	return obj
}

func setError(dbObj *v1alpha1.Database, error error) *v1alpha1.Database {
	dbObjClone := dbObj.DeepCopy()
	var errorStatus string
	if error != nil {
		errorStatus = error.Error()
	}
	dbObjClone.Status.Error = errorStatus
	return dbObjClone
}

type state string

const (
	disconnected state = "Not connected"
	connected    state = "Connected"
)

func setState(dbObj *v1alpha1.Database, state state) *v1alpha1.Database {
	dbObjClone := dbObj.DeepCopy()
	dbObjClone.Status.State = string(state)
	return dbObjClone
}
