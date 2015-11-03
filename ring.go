package libring

import (
	"fmt"

	"hash/fnv"

	"github.com/dgryski/go-jump"
	"github.com/hashicorp/serf/serf"
)

// Handles resolving a key/partition to an array of serf members such that the
// mapping is maximally stable across cluster membership changes.
//
// Used both for fetching serf members and for detecting changes before & after
// the cluster memberhsip changes.
type ring struct {
	distribution   DistributionMethod
	partitionCount uint
	members        []*serf.Member
}

func (r ring) String() string {
	memberNames := make([]string, len(r.members), len(r.members))
	for i, member := range r.members {
		memberNames[i] = member.Name
	}
	return fmt.Sprintf("%v", memberNames)
}

func (r ring) membersForKey(key string) []serf.Member {
	partition := r.partitionForKey(key)
	return r.membersForPartition(partition)
}

func (r ring) membersForPartition(partition uint) (members []serf.Member) {
	for replica := 0; replica < len(r.members); replica++ {
		if member := r.member(partition, uint(replica)); member != nil {
			members = append(members, *member)
		}
	}

	return members
}

func (r ring) member(partition uint, replica uint) *serf.Member {
	if len(r.members) == 0 {
		return nil
	}

	if uint(len(r.members)) <= replica {
		return nil
	}

	var rotation uint
	switch r.distribution {
	case ConsistentHashing:
		rotation = uint(jump.Hash(uint64(partition), len(r.members)))
	case Uniform:
		rotation = partition % uint(len(r.members))
	}

	index := (rotation + replica) % uint(len(r.members))
	return r.members[index]

}

func (r ring) partitionForKey(key string) uint {
	hasher := fnv.New64a()
	hasher.Write([]byte(key))
	keyHash := hasher.Sum64()

	return uint(keyHash) % r.partitionCount
}
