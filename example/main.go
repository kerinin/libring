package main

import (
	"os"
	"regexp"

	"github.com/kerinin/libring"
)

func main() {
	config := libring.Config{
		WatchTags: map[string]*regexp.Regexp{"ring": regexp.MustCompile(`/.*/`)},
		Partitions: 8,
		Redundancy: 2,
		Acquisitions: make(chan libring.AcquireEvent),
		Releases: make(chan libring.ReleaseEvent),
	}

	if os.Getenv("BOOTSTRAP_HOST") != "" {
		config.BootstrapHosts = []string{os.Getenv("BOOTSTRAP_HOST")}
	}

	cluster := libring.NewCluster(config)

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

	// If this host should be part of the cluster, update its tags.
	cluster.SetTags(map[string]string{"ring": "1"})

	cluster.Run()

	<-cluster.Exit
}
