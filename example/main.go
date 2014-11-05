package main

import (
	"time"
	"os"
	"regexp"

	"github.com/kerinin/libring"
)

func main() {
	config := libring.Config{
		WatchTags: map[string]*regexp.Regexp{"ring": regexp.MustCompile(`active`)},
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

	go cluster.Run()

	time.Sleep(2 * time.Second)
	cluster.SetTags(map[string]string{"ring": "active"})

	time.Sleep(2 * time.Second)
	cluster.SetTags(map[string]string{"ring": "leaving"})

	time.Sleep(2 * time.Second)
	cluster.Stop()

	logger.Info("Exiting")
}
