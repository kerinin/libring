package libring

import (
	"github.com/hashicorp/serf/serf"
)

type ReleaseEvent struct {
	Partition uint
	Replica uint
	To        *serf.Member
	SerfEvent *serf.Event
}
