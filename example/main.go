package main

import (
	"os"
	"regexp"
	"time"

	"github.com/kerinin/libring"
)

func main() {
	// Setup the config.  Could also use libring.DefaultConfig()
	config := libring.Config{
		WatchTags:    map[string]*regexp.Regexp{"ring": regexp.MustCompile(`active`)},
		Partitions:   8,
		Redundancy:   2,
		Acquisitions: make(chan libring.AcquireEvent),
		Releases:     make(chan libring.ReleaseEvent),
	}

	// See if there's an existing Serf clsuter to join
	if os.Getenv("BOOTSTRAP_HOST") != "" {
		config.BootstrapHosts = []string{os.Getenv("BOOTSTRAP_HOST")}
	}

	// Create the cluster
	cluster := libring.NewCluster(config)

	// Start listening for cluster events
	go func() {
		for acquisition := range config.Acquisitions {
			if acquisition.From == nil {
				logger.Info("Partition %d/%d acquired", acquisition.Partition, acquisition.Replica)
			} else {
				logger.Info("Partition %d/%d acquired from %s", acquisition.Partition, acquisition.Replica, acquisition.From.Name)
			}
		}
	}()
	go func() {
		for release := range config.Releases {
			if release.To == nil {
				logger.Info("Partition %d/%d released", release.Partition, release.Replica)
			} else {
				logger.Info("Partition %d/%d released to %s", release.Partition, release.Replica, release.To.Name)
			}
		}
	}()

	// Start the cluster
	go cluster.Run()

	// Wait a bit for cluster state to become consistent, then set serf tags
	// This will add the local node to the ring
	time.Sleep(2 * time.Second)
	cluster.SetTags(map[string]string{"ring": "active"})

	// This will remove the local node from the ring (see the regex above), but
	// keep the serf client active.  This could be useful for doing cleanup.
	time.Sleep(2 * time.Second)
	cluster.SetTags(map[string]string{"ring": "leaving"})

	// Leave the cluster
	time.Sleep(2 * time.Second)
	cluster.Stop()

	logger.Info("Exiting")
}
