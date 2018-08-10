package db

import (
	"github.com/golang/glog"
	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
	"fmt"
	"strings"
	"time"
)

type PostgreSQLServer struct {
	Host       string
	Port       int
	User       string
	Password   string
	EnableSSL  bool
	Production bool
}

// Postgres cannot handle db names with '-' or '.'. We replace them with '_'.
// This will lead to collisions. But documenting this should be enough.
// No sane person will want to use "my-db" and "my_db" at the same time...
func stripInvalidCharacters(dbName string) string {
	newName := strings.Replace(dbName, "-", "_", -1)
	return strings.Replace(newName, ".", "_", -1)
}

func (pg PostgreSQLServer) Create(dbName, user, password string) error {
	dbName = stripInvalidCharacters(dbName)
	glog.Infof("Deploying Postgres Database: %s to Server %s", dbName, pg.Host)
	var sslmode string
	if pg.EnableSSL {
		sslmode = "verify-full"
	} else {
		sslmode = "disable"
	}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s",
		pg.Host, pg.Port, pg.User, pg.Password, sslmode)
	db, err := sqlx.Connect("postgres", psqlInfo)
	if err != nil {
		glog.Errorf("Error Connecting to server: %s", err.Error())
		return err
	}
	defer db.Close()

	// Check if user exists
	userExitsQuerry := "SELECT 1 FROM pg_roles WHERE rolname=$1 "
	userRow, err := db.Query(userExitsQuerry, user)
	if err != nil {
		glog.Errorf("Error reading Database: %s", err.Error())
		return err
	}
	defer userRow.Close()

	// Create the user if not exits
	if !userRow.Next() {
		userCreation := fmt.Sprintf("CREATE ROLE \"%s\" LOGIN PASSWORD '%s'", user, password)
		_, err = db.Exec(userCreation)
		if err != nil {
			glog.Errorf("Error Creating User: %s", err.Error())
			return err
		}
	}

	// Check if db exits
	dbExitsQuerry := "SELECT 1 FROM pg_database WHERE datname=$1"
	dbRow, err := db.Query(dbExitsQuerry, dbName)
	if err != nil {
		glog.Errorf("Error reading Database: %s", err.Error())
		return err
	}
	defer dbRow.Close()

	// Create db if not exits
	if !dbRow.Next() {
		// Kuberenetes resource names may only contain lower case letters, . and -.
		// This means we don't have to worry about sql injection, although sqlx is not injection save when
		// creating dbs
		dbCreation := fmt.Sprintf("CREATE DATABASE %s", dbName)
		_, err = db.Exec(dbCreation)
		if err != nil {
			glog.Errorf("Error Creating Database: %s", err.Error())
			return err
		}

	}

	// Grant access to db
	grantAccess := fmt.Sprintf("GRANT ALL ON DATABASE \"%s\" TO \"%s\"", dbName, user)
	_, err = db.Exec(grantAccess)
	if err != nil {
		glog.Errorf("Error Granting access to Database: %s", err.Error())
		return err
	}

	// Check if user can connect
	if !pg.Ping(dbName, user, password) {
		glog.Errorf("Error User cannot connect to Database: %s", err.Error())
		return fmt.Errorf("user cannot connect to Database; %s", err.Error())
	}

	return nil
}

func (pg PostgreSQLServer) Delete(dbName string) error {
	dbName = stripInvalidCharacters(dbName)
	glog.Infof("Dropping Postgres Database: %s on Server %s", dbName, pg.Host)
	var sslmode string
	if pg.EnableSSL {
		sslmode = "verify-full"
	} else {
		sslmode = "disable"
	}
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s",
		pg.Host, pg.Port, pg.User, pg.Password, sslmode)
	db, err := sqlx.Connect("postgres", psqlInfo)
	if err != nil {
		glog.Errorf("Error Connecting to server: %s", err.Error())
		return err
	}
	defer db.Close()

	if !pg.Production {
		dbDeletion := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)
		_, err = db.Exec(dbDeletion)
		if err != nil {
			glog.Errorf("Error Dropping Database: %s", err.Error())
			return err
		}
	} else {
		// If we are in a prod environment we don't want to erase the database.  We will rename it to
		// DELETED_<timestam>_<dbName>. We add a timestamp to prevent overwriting.

		// Check if db exits
		dbExitsQuerry := "SELECT 1 FROM pg_database WHERE datname=$1"
		dbRow, err := db.Query(dbExitsQuerry, dbName)
		if err != nil {
			glog.Errorf("Error reading Database: %s", err.Error())
			return err
		}
		defer dbRow.Close()

		// Rename db if exits
		if dbRow.Next() {
			now := time.Now()
			dbDeletedName := fmt.Sprintf("DELETED_%s_%s", now.Format("20060102T150405"), dbName)
			dbRename := fmt.Sprintf("ALTER DATABASE %s RENAME TO %s", dbName, dbDeletedName)
			_, err = db.Exec(dbRename)
			if err != nil {
				glog.Errorf("Error Dropping Database: %s", err.Error())
				return err
			}
		}
	}
	return nil
}

func (pg PostgreSQLServer) Ping(dbName, user, password string) bool {
	dbName = stripInvalidCharacters(dbName)
	var sslmode string
	if pg.EnableSSL {
		sslmode = "verify-full"
	} else {
		sslmode = "disable"
	}
	psqlInfo := fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		pg.Host, pg.Port, dbName, user, password, sslmode)
	db, err := sqlx.Connect("postgres", psqlInfo)
	if err != nil {
		return false
	}
	defer db.Close()
	return true
}
