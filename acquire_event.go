package libring

import (
	"github.com/hashicorp/serf/serf"
)

type AcquireEvent struct {
	Partition uint
	Replica uint
	From      *serf.Member
	SerfEvent *serf.Event
}
