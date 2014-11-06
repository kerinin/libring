package libring

import (
	"github.com/hashicorp/serf/serf"
)

// Represents a partition replica being removed from management by the local node.
// Can be triggered by the local node shutting down, or by nodes entering/leaving
// the cluster causing partitions to be rebalanced across the active nodes.
type ReleaseEvent struct {
	// The partition which was released.  Partition values range from 0 to the
	// value of 'Partitions' in the cluster config
	Partition uint

	// The partition replica which was release.  Replica values range from 0 to
	// the value of 'Redundancy' in the cluster config
	Replica uint

	// The serf member that Partition was released to.  Can be nil if the
	// partition/replica is no longer exists in the cluster (ie if the number of
	// hosts is less than config.Redundancy)
	To *serf.Member

	// The serf event that triggered the partition to be released.  See
	// http://godoc.org/github.com/hashicorp/serf/serf#Event for more information
	SerfEvent *serf.Event
}
