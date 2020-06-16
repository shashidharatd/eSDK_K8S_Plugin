package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	log "github.com/golang/glog"
	"google.golang.org/grpc"

	"github.com/Huawei/eSDK_K8S_Plugin/pkg/backend"
	"github.com/Huawei/eSDK_K8S_Plugin/pkg/driver"
)

const (
	configFile        = "/etc/huawei/csi.json"
	controllerLogFile = "huawei-csi-controller"
	nodeLogFile       = "huawei-csi-node"
	csiLogFile        = "huawei-csi"

	csiVersion        = "2.2.6"
	defaultDriverName = "csi.huawei.com"
)

var (
	endpoint           = flag.String("endpoint", "", "CSI endpoint")
	controller         = flag.Bool("controller", false, "Run as a controller service")
	controllerFlagFile = flag.String("controller-flag-file", "",
		"The flag file path to specify controller service. Privilege is higher than controller")
	driverName = flag.String("driver-name", defaultDriverName, "CSI driver name")

	config CSIConfig
)

type CSIConfig struct {
	Backends []map[string]interface{} `json:"backends"`
}

func init() {
	flag.Parse()

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Read config file %s error: %v", configFile, err)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Unmarshal config file %s error: %v", configFile, err)
	}

	if len(config.Backends) <= 0 {
		log.Fatalf("Must configure at least one backend")
	}
}

func updateBackendCapabilities() {
	err := backend.SyncUpdateCapabilities()
	if err != nil {
		log.Fatalf("Update backend capabilities error: %v", err)
	}

	if len(*controllerFlagFile) > 0 || *controller {
		ticker := time.NewTicker(time.Minute)
		for range ticker.C {
			backend.AsyncUpdateCapabilities(*controllerFlagFile)
		}
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("Runtime error caught in main routine: %v", r)
			log.Errorf("%s", debug.Stack())
		}

		log.Flush()
	}()

	err := backend.RegisterBackend(config.Backends)
	if err != nil {
		log.Fatalf("Register backends error: %v", err)
	}

	endpointDir := filepath.Dir(*endpoint)
	_, err = os.Stat(endpointDir)
	if err != nil && os.IsNotExist(err) {
		os.Mkdir(endpointDir, 0755)
	} else {
		_, err := os.Stat(*endpoint)
		if err == nil {
			log.Infof("Gonna remove old sock file %s", *endpoint)
			os.Remove(*endpoint)
		}
	}

	listener, err := net.Listen("unix", *endpoint)
	if err != nil {
		log.Fatalf("Listen on %s error: %v", *endpoint, err)
	}

	d := driver.NewDriver(*driverName, csiVersion)
	server := grpc.NewServer()

	csi.RegisterIdentityServer(server, d)
	csi.RegisterControllerServer(server, d)
	csi.RegisterNodeServer(server, d)

	go updateBackendCapabilities()

	log.Infof("Starting Huawei CSI driver, listening on %s", *endpoint)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Start Huawei CSI driver error: %v", err)
	}
}
