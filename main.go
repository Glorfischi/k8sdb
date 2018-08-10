package main

import (
	"flag"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/golang/glog"
	"github.com/spf13/viper"
	"path/filepath"
	"os"
	"github.com/glorfischi/k8sdb/pkg/db"
	"github.com/glorfischi/k8sdb/pkg/controller"
)

type Configuration struct {
	Postgres map[string]db.PostgreSQLServer
	Test     map[string]db.MockDbServer
}

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	var configuration Configuration

	if err := viper.ReadInConfig(); err != nil {
		glog.Fatalf("Error reading config file, %s", err)
	}
	err := viper.Unmarshal(&configuration)
	if err != nil {
		glog.Fatalf("unable to decode into struct, %v", err)
	}

	// adding test db server for debugging purposes.
	dbServers := map[string]db.DatabaseServer{"test": db.MockDbServer{}}

	// adding configured dbs for all supported database types
	// adding postgres dbs
	for k, v := range configuration.Postgres {
		dbServers[k] = v
	}


	// get kube config
	var kubeconfig *string
	if home := os.Getenv("HOME"); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"),
			"(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := make(chan struct{})

	cfg, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		glog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	controller, err := controller.NewController(cfg, dbServers)
	if err != nil {
		glog.Fatalf("Error building contoller: %s", err.Error())
	}

	if err = controller.Run(stopCh); err != nil {
		glog.Fatalf("Error running controller: %s", err.Error())
	}
	glog.Flush()
}
