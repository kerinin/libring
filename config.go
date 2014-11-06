package libring

import (
	"os"
	"regexp"
)

// Configuration values for the libring Cluster
type Config struct {
	// Specify a set of tag/values which must be present on a Serf member to be
	// treated as part of the cluster.  Allows multiple clusters to share members,
	// and allows members to communicate about their current state
	WatchTags map[string]*regexp.Regexp

	// Join the Serf cluster that these hosts are part of.  Can be pointed at a
	// load balancer if you hostnames are dynamically assigned.
	BootstrapHosts []string

	// Specifies the hostname to use for the local node.  Defaults to the
	// OS-provided value
	HostName string

	// The number of partitions to divide the keyspace into.  This value should be
	// an order of maginitude larger than the number of members you expect to
	// participate in the cluster.  Larger values increase the amount of metadata
	// the cluster must track, but smaller values limit the scalability of the
	// cluster.  The partition count is an upper-bound on the number of hosts
	// which can participate in a cluster
	Partitions uint

	// Partitions will be assigned to this many hosts at any given point in time.
	// This represents a lower bound on the number of hosts you should have
	// running at any point in time.
	Redundancy uint

	// Channels for receiving notifications when partitions are assigned to the
	// local machine or removed from the local machine.  Events contain the partition
	// identifier, the 'other' Member, and the serf Event which triggered the
	// partition to be reassigned.
	Acquisitions chan AcquireEvent
	Releases     chan ReleaseEvent
}

// Returns a Config with sane default values.
func DefaultConfig() Config {
	hostname, _ := os.Hostname()

	return Config{
		HostName:   hostname,
		Partitions: 512,
		Redundancy: 3,
	}
}
