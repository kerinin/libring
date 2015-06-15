package libring

import (
	"github.com/hashicorp/serf/serf"
)

// EventType desribes if an event is an acquisition or release
type EventType int

const (
	// Acquisition reflects a partition which has been newly assigned to "this" member
	Acquisition EventType = iota
	// Release reflects a partition which has been removed from "this" member
	Release
)

// Event represents a partition replica being assigned to the local node for management.
// Can be triggered by a node spinning up in a new cluster, or a node
// entering/leaving the cluster and partitions being rebalanced across the remaining
// nodes.
type Event struct {
	// The event type
	Type EventType

	// The partition which was acquired.  Partition values range from 0 to the
	// value of 'Partitions' in the cluster config
	Partition uint

	// The partition replica which was acquired.  Replica values range from 0 to
	// the value of 'Redundancy' in the cluster config
	Replica uint

	// The serf member that Partition was acquired from.  Can be nil if the
	// partition/replica didn't previously exist in the cluster (ie if the
	// cluster is being bootstrapped) or if the EventType is Release
	From *serf.Member

	// The serf member that Partition was released to.  Can be nil if the
	// partition/replica is no longer exists in the cluster (ie if the number of
	// hosts is less than config.Redundancy) or if the EventType is Acquisition
	To *serf.Member

	// The serf event that triggered the partition to be released.  See
	// http://godoc.org/github.com/hashicorp/serf/serf#Event for more information
	SerfEvent *serf.Event
}
