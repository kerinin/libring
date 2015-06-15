package main

import (
	"os"
	"regexp"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/hashicorp/serf/serf"
	"github.com/kerinin/libring"
)

func main() {
	// Setup the config.  Could also use libring.DefaultConfig()
	config := libring.DefaultConfig()
	config.WatchTags = map[string]*regexp.Regexp{"ring": regexp.MustCompile(`active`)}
	config.Partitions = 8
	config.Redundancy = 2
	config.Events = make(chan libring.Event)
	config.SerfEvents = make(chan serf.Event)

	// See if there's an existing Serf clsuter to join
	if os.Getenv("BOOTSTRAP_HOST") != "" {
		config.BootstrapHosts = []string{os.Getenv("BOOTSTRAP_HOST")}
	}

	// Create the cluster
	cluster, err := libring.NewCluster(config)
	if err != nil {
		logrus.Error("Unable to create cluster: %v", err)
		return
	}

	// Start listening for cluster events
	go func() {
		for event := range config.Events {
			switch event.Type {
			case libring.Acquisition:
				if event.From == nil {
					logrus.Infof("Partition %d/%d acquired", event.Partition, event.Replica)
				} else {
					logrus.Infof("Partition %d/%d acquired from %s", event.Partition, event.Replica, event.From.Name)
				}
			case libring.Release:
				if event.To == nil {
					logrus.Infof("Partition %d/%d released", event.Partition, event.Replica)
				} else {
					logrus.Infof("Partition %d/%d released to %s", event.Partition, event.Replica, event.To.Name)
				}
			}
		}
	}()
	go func() {
		for event := range config.SerfEvents {
			logrus.Infof("Serf fired event: %v", event)
		}
	}()

	// Start the cluster
	go cluster.Run()

	// Wait a bit for cluster state to become consistent, then set serf tags
	// This will add the local node to the ring
	time.Sleep(2 * time.Second)
	cluster.Serf.SetTags(map[string]string{"ring": "active"})

	// This will remove the local node from the ring (see the regex above), but
	// keep the serf client active.  This could be useful for doing cleanup.
	time.Sleep(2 * time.Second)
	cluster.Serf.SetTags(map[string]string{"ring": "leaving"})

	// Leave the cluster
	time.Sleep(2 * time.Second)
	cluster.Stop()

	logrus.Info("Exiting")
}
