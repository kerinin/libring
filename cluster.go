package libring

import (
	"sort"
	"sync"

	"github.com/hashicorp/serf/serf"
)

type Cluster struct {
	Exit chan bool
	config      Config
	memberMap   map[string]*serf.Member
	ring *Ring
	memberMutex sync.Mutex
	serf        *serf.Serf
	serfEvents  chan serf.Event
}

func NewCluster(config Config) *Cluster {
	memberMap := make(map[string]*serf.Member)
	ring := &Ring{members: make([]*serf.Member, 0, 0)}
	memberMutex := sync.Mutex{}
	serfEvents := make(chan serf.Event, 256)

	serfConfig := serf.DefaultConfig()
	serfConfig.EventCh = serfEvents
	nodeSerf, _ := serf.Create(serfConfig)
	exit := make(chan bool)

	return &Cluster{
		Exit: exit,
		config:      config,
		memberMap:   memberMap,
		ring: ring,
		memberMutex: memberMutex,
		serfEvents:  serfEvents,
		serf:        nodeSerf,
	}
}

func (c *Cluster) SetTags(tags map[string]string) {
}

func (c *Cluster) Run() {
	go func() {
		for e := range c.serfEvents {
			c.handleSerfEvent(e)
		}
	}()

	c.joinSerfCluster()
}

func (c *Cluster) Stop() {
}

func (c *Cluster) MembersForKey(key string) chan *serf.Member {
	return c.ring.MembersForKey(key)
}

func (c *Cluster) MembersForPartition(partition uint) chan *serf.Member {
	return c.ring.MembersForPartition(partition)
}

func (c *Cluster) joinSerfCluster() {
	if len(c.config.BootstrapHosts) > 0 {
		c.serf.Join(c.config.BootstrapHosts, true)
	}
}

func (c *Cluster) handleRingChange(event *serf.Event, old_ring *Ring, new_ring *Ring) {
	for partition := uint(0); partition < c.config.Partitions; partition++ {
		old_members := old_ring.MembersForPartition(partition)
		new_members := new_ring.MembersForPartition(partition)

		if c.config.Releases != nil {
			for replica := uint(0); replica < c.config.Redundancy; replica++ {

				// If partition/replica used to be owned by the local node
				old_member, ok := <-old_members
				if !ok {
					break
				}

				if old_member != nil && old_member.Name == c.serf.LocalMember().Name {
					// ...but isn't any longer
					new_member := new_ring.Member(partition, replica)
					if new_member == nil || new_member.Name != c.serf.LocalMember().Name {
						event := ReleaseEvent{
							Partition: partition,
							Replica: replica,
							To: new_member,
							SerfEvent: event,
						}

						c.config.Releases <-  event
					}
				}
			}
		}

		if c.config.Acquisitions != nil {
			for replica := uint(0); replica < c.config.Redundancy; replica++ {

				// If partition/replica is owned by the local node
				new_member, ok := <-new_members
				if !ok {
					break
				}

				if new_member != nil && new_member.Name == c.serf.LocalMember().Name {
					// ...but didn't used to be
					old_member := old_ring.Member(partition, replica)
					if old_member == nil || old_member.Name != c.serf.LocalMember().Name {
						event := AcquireEvent{
							Partition: partition,
							Replica: replica,
							From: old_ring.Member(partition, replica),
							SerfEvent: event,
						}

						c.config.Acquisitions <- event
					}
				}
			}
		}
	}
}

func (c *Cluster) addEventMembers(e serf.Event) {
	c.memberMutex.Lock()
	old_ring := *c.ring
	for _, member := range e.(serf.MemberEvent).Members {
		c.memberMap[member.Name] = &member
	}
	c.recomputeRing()
	new_ring := *c.ring // caching this to reduce time inside the mutex
	c.memberMutex.Unlock()
	c.handleRingChange(&e, &old_ring, &new_ring)
}

func (c *Cluster) updateEventMembers(e serf.Event) {
	c.memberMutex.Lock()
	old_ring := *c.ring
	for _, member := range e.(serf.MemberEvent).Members {
		c.memberMap[member.Name] = &member
	}
	c.recomputeRing()
	new_ring := *c.ring // caching this to reduce time inside the mutex
	c.memberMutex.Unlock()
	c.handleRingChange(&e, &old_ring, &new_ring)
}

func (c *Cluster) removeEventMembers(e serf.Event) {
	c.memberMutex.Lock()
	old_ring := *c.ring
	for _, member := range e.(serf.MemberEvent).Members {
		delete(c.memberMap, member.Name)
	}
	c.recomputeRing()
	new_ring := *c.ring // caching this to reduce time inside the mutex
	c.memberMutex.Unlock()
	c.handleRingChange(&e, &old_ring, &new_ring)
}

func (c *Cluster) handleSerfEvent(e serf.Event) {
	switch e.EventType() {
	case serf.EventMemberJoin:
		logger.Debug("Handling member join event", e.(serf.MemberEvent).Members)
		go c.addEventMembers(e)

	case serf.EventMemberLeave:
		logger.Debug("Handling graceful member exit event", e.(serf.MemberEvent))
		go c.removeEventMembers(e)

	case serf.EventMemberFailed:
		logger.Debug("Handling unresponsive member event", e.(serf.MemberEvent))
		go c.updateEventMembers(e)

	case serf.EventMemberUpdate:
		logger.Debug("Handling member metadata update event", e.(serf.MemberEvent))
		go c.updateEventMembers(e)

	case serf.EventMemberReap:
		logger.Debug("Handling forced member exit event", e.(serf.MemberEvent))
		go c.removeEventMembers(e)

	default:
		logger.Warning("Unhandled Serf event: %#v", e)
	}
}

// NOTE: Should probably return data for acquire/release streams
func (c *Cluster) recomputeRing() {
	keys := make([]string, len(c.memberMap), len(c.memberMap))
	i := 0
	for k, _ := range c.memberMap {
		keys[i] = k
		i++
	}
	sort.StringSlice(keys).Sort()

	members := make([]*serf.Member, len(c.memberMap), len(c.memberMap))
	for i, k := range keys {
		members[i], _ = c.memberMap[k]
	}

	c.ring = &Ring{members: members}
	logger.Debug("Recomputed ring: %v", c.ring)
}
