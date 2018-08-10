package controller

import (
	"github.com/glorfischi/k8sdb/pkg/db"
	"github.com/glorfischi/k8sdb/pkg/apis/k8sdb/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"fmt"
	"k8s.io/api/core/v1"
)

func (c *Controller) handleEvent(ev event) error {
	// Get the new Db resource with this namespace/name
	newObj, exists, err := c.informer.GetIndexer().GetByKey(ev.newKey)
	if err != nil {
		return err
	}

	// This object does not exits anymore. Just ignore it.
	if !exists {
		return nil
	}

	db, ok := newObj.(*v1alpha1.Database)
	if !ok {
		return nil
	}

	// get correct DBServer.
	dbType := db.Spec.Type
	dbServer, ok := c.dbServers[dbType]
	if !ok {
		error := fmt.Errorf("dbType: '%v' not supported", dbType)
		dbClone := setError(db, error)
		updateDbObj(c.dbClientset, dbClone)
		return error
	}

	var dbClone *v1alpha1.Database
	var secretClone *v1.Secret
	switch ev.eventType {
	case add:
		dbClone, secretClone, err = c.handleAdd(dbServer, db)
	case update:
		dbClone, secretClone, err = c.handleUpdate(dbServer, nil, db)
	}

	if dbClone != nil {
		db = dbClone
	}

	if err != nil {
		db = setError(db, err)
		db = setState(db, disconnected)
	} else {
		// If there was no error, we check if we have a connection to the db and remove any error if we do.
		secret, err := c.kubeClientset.CoreV1().Secrets(db.Namespace).Get(db.Spec.Credentials, metav1.GetOptions{})
		if err != nil {
			err = fmt.Errorf("Secret '%s' not found", db.Spec.Credentials)
		}
		name := db.Name
		user := string(secret.Data["user"])
		pw := string(secret.Data["password"])

		if dbServer.Ping(name, user, pw) {
			db = setError(db, nil)
			db = setState(db, connected)
		} else {
			db = setError(db, err)
			db = setState(db, disconnected)
		}
	}

	if e := updateDbObj(c.dbClientset, db); e != nil {
		return e
	}
	if secretClone != nil {
		e := updateSecret(c.kubeClientset, secretClone)
		if e != nil {
			return e
		}
	}

	return err
}

func (c *Controller) handleAdd(dbServer db.DatabaseServer, db *v1alpha1.Database) (*v1alpha1.Database, *v1.Secret, error) {
	secret, err := c.kubeClientset.CoreV1().Secrets(db.Namespace).Get(db.Spec.Credentials, metav1.GetOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("Secret '%s' not found", db.Spec.Credentials)
	}
	name := db.Name
	user := string(secret.Data["user"])
	pw := string(secret.Data["password"])

	dbClone := db.DeepCopy()
	secretClone := secret.DeepCopy()

	// Create db and add finalizer if there is not already one
	if !hasFinalizer(db, dbFinalizer) {
		if err := dbServer.Create(name, user, pw); err != nil {
			return nil, nil, err
		}
		addFinalizer(dbClone, dbFinalizer)
		addFinalizer(secretClone, dbFinalizer + "-" + db.Namespace + "-" + db.Name)

	}
	return dbClone, secretClone, nil
}

func (c *Controller) handleUpdate(dbServer db.DatabaseServer, oldDb, newDb *v1alpha1.Database) (*v1alpha1.Database, *v1.Secret, error) {
	// The database is marked for deletion
	if newDb.ObjectMeta.DeletionTimestamp != nil {
		secret, err := c.kubeClientset.CoreV1().Secrets(newDb.Namespace).Get(newDb.Spec.Credentials, metav1.GetOptions{})
		if err != nil {
			return nil, nil, err
		}
		dbClone := newDb.DeepCopy()
		secretClone := secret.DeepCopy()
		// If finalizer is removed, then we already processed the delete update, so just return
		if hasFinalizer(newDb, dbFinalizer) {
			name := newDb.Name

			if err := dbServer.Delete(name); err != nil {
				return nil, nil, err
			}
			deleteAllFinalizers(dbClone)
			deleteFinalizer(secretClone, dbFinalizer + "-" + newDb.Namespace + "-" + newDb.Name)
		}
		return dbClone, secretClone, nil
	}
	return nil, nil, nil
}
