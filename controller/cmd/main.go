package main

import (
	"context"
	"flag"
	"time"

	"controller/pkg/p4switch"
	"controller/pkg/server"

	log "github.com/sirupsen/logrus"
)

const (
	defaultPort     = 50050
	defaultAddr     = "127.0.0.1"
	defaultWait     = 250 * time.Millisecond
	packetCounter   = "MyIngress.port_packets_in"
	packetCountWarn = 20
	packetCheckRate = 5 * time.Second
	p4topology      = "../config/topology.json"
)

func main() {

	// Inizializza variabili "flag" che vengono passate come argomento

	var nDevices int
	flag.IntVar(&nDevices, "n", 1, "Number of devices")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose mode with debug log messages")
	var trace bool
	flag.BoolVar(&trace, "trace", false, "Enable trace mode with log messages")
	var configName string
	flag.StringVar(&configName, "config", "../config/config.json", "Program name")
	var topologyName string
	flag.StringVar(&topologyName, "topology", "", "Topology name")
	var certFile string
	flag.StringVar(&certFile, "cert-file", "", "Certificate file for tls")
	flag.Parse()

	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	if trace {
		log.SetLevel(log.TraceLevel)
	}
	log.Infof("Starting %d devices", nDevices)

	ctx, cancel := context.WithCancel(context.Background())
	switchs := make([]*p4switch.GrpcSwitch, 0, nDevices)

	for i := 0; i < nDevices; i++ {
		sw := p4switch.CreateSwitch(uint64(i+1), configName, 3, certFile)

		if err := sw.RunSwitch(ctx); err != nil {

			sw.GetLogger().Errorf("Cannot start")
			log.Errorf("%v", err)

		} else {
			switchs = append(switchs, sw)
		}

	}
	if len(switchs) == 0 {
		log.Info("No switches started")
		return
	}

	server.StartServer(switchs, topologyName)
	cancel()
}
