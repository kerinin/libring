package libring

import (
	"io"
	"os"
	"regexp"

	"github.com/hashicorp/serf/serf"
)

type DistributionMethod uint8

const (
	ConsistentHashing DistributionMethod = iota
	Uniform
)

// Config stores configuration values for a libring Cluster
type Config struct {
	// Specify a set of tag/values which must be present on a Serf member to be
	// treated as part of the cluster.  Allows multiple clusters to share members,
	// and allows members to communicate about their current state
	// Leave empty to use all members of the serf cluster
	WatchTags map[string]*regexp.Regexp

	// Join the Serf cluster that these hosts are part of.  Can be pointed at a
	// load balancer if you hostnames are dynamically assigned.
	BootstrapHosts []string

	// PartitionDistribution is the method used to distribute partitions across
	// hosts.
	//
	// * ConsistentHashing (Default) - Minimizes the number of partitions which
	//   need to be moved as the ring grows or shrinks. However, if the number
	//   partitions is small they may be distributed poorly across hosts.
	//
	// * Uniform - Distributes partitions evenly across the hosts at the cost
	//   of having to move more partitions when the ring grows or shrinks. May
	//   be desirable if moving partitions is cheap and ConsistentHashing is
	//   distributing partitions poorly.
	PartitionDistribution DistributionMethod

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

	// The serf client will be created with this configuration, so if you need to
	// do anything unusual you can set it here.  Note that libring will specify
	// the EventCh, specifying it in this config is an error.  (If you need to
	// handle raw serf events, you can provide a channel to SerfEvents below)
	SerfConfig *serf.Config

	// If provided, serf events will be pushed to this channel *after* they have
	// been processed by libring.  Note that the same warning applies here as
	// to serf.Config.EventCh: "Care must be taken that this channel doesn't
	// block, either by processing the events quick enough or buffering the
	// channel"
	SerfEvents chan serf.Event

	// Channel for receiving notifications when partitions are assigned to the
	// local machine or removed from the local machine.  Events contain the partition
	// identifier, the 'other' Member, and the serf Event which triggered the
	// partition to be reassigned.
	Events chan Event

	// LogOutput is the location to write logs to. If this is not set,
	// logs will go to stderr.
	LogOutput io.Writer

	// LogLevel: Panic = 0, Fatal, Error, Warn, Info, Debug = 5
	LogLevel uint8
}

// DefaultConfig returns a Config with sane default values.
func DefaultConfig() Config {
	return Config{
		PartitionDistribution: ConsistentHashing,
		Partitions:            512,
		Redundancy:            3,
		SerfConfig:            serf.DefaultConfig(),
		Events:                make(chan Event),
		LogOutput:             os.Stderr,
		LogLevel:              2,
	}
}
