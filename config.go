package libring

import (
	"os"
	"regexp"
)

type Config struct {
	WatchTags      map[string]*regexp.Regexp
	BootstrapHosts []string
	HostName       string
	Partitions     uint
	Redundancy     uint
	Acquisitions   chan AcquireEvent
	Releases       chan ReleaseEvent
}

func DefaultConfig() Config {
	hostname, _ := os.Hostname()

	return Config{
		HostName: hostname,
		Partitions: 512,
		Redundancy: 3,
	}
}
