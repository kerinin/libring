package libring

import (
	"fmt"

	"hash/fnv"

	"github.com/dgryski/go-jump"
	"github.com/hashicorp/serf/serf"
)

type Ring struct {
	members     []*serf.Member
}

func (r Ring) String() string {
	member_names := make([]string, len(r.members), len(r.members))
	for i, member := range r.members {
		member_names[i] = member.Name
	}
	return fmt.Sprintf("%v", member_names)
}

func (r Ring) MembersForKey(key string) chan *serf.Member {
	partition := r.partitionForKey(key)
	return r.MembersForPartition(partition)
}

func (r Ring) MembersForPartition(partition uint) chan *serf.Member {
	outCh := make(chan *serf.Member)

	if len(r.members) == 0 {
		close(outCh)
		return outCh
	}

	go func() {
		for replica := 0; replica < len(r.members); replica++ {
			if member := r.Member(partition, uint(replica)); member != nil {
				outCh <- member
			}
		}
		close(outCh)
	}()

	return outCh
}

func (r Ring) Member(partition uint, replica uint) *serf.Member {
	if len(r.members) == 0 {
		return nil 
	}
	if uint(len(r.members)) <= replica {
		return nil
	}

	rotation := uint(jump.Hash(uint64(partition), len(r.members)))

	index := (rotation + replica) % uint(len(r.members))

	return r.members[index]
}

func (r Ring) partitionForKey(key string) uint {
	hasher := fnv.New64a()
	hasher.Write([]byte(key))
	key_hash := hasher.Sum64()

	return uint(jump.Hash(key_hash, len(r.members)))
}
