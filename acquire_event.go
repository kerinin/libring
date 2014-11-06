package libring

import (
	"github.com/hashicorp/serf/serf"
)

// Represents a partition replica being assigned to the local node for management.
// Can be triggered by a node spinning up in a new cluster, or a node
// entering/leaving the cluster and partitions being rebalanced across the remaining
// nodes.
type AcquireEvent struct {
	// The partition which was acquired.  Partition values range from 0 to the
	// value of 'Partitions' in the cluster config
	Partition uint

	// The partition replica which was acquired.  Replica values range from 0 to
	// the value of 'Redundancy' in the cluster config
	Replica uint

	// The serf member that Partition was acquired from.  Can be nil if the
	// partition/replica didn't previously exist in the cluster (ie if the
	// cluster is being bootstrapped)
	From *serf.Member

	// The serf event that triggered the partition to be released.  See
	// http://godoc.org/github.com/hashicorp/serf/serf#Event for more information
	SerfEvent *serf.Event
}
