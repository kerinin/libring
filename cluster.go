package libring

import (
	"sort"
	"sync"

	"github.com/hashicorp/serf/serf"
)

// The primary libring interface
type Cluster struct {
	exit        chan bool
	config      Config
	memberMap   map[string]*serf.Member
	ring        *Ring
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
		exit:        exit,
		config:      config,
		memberMap:   memberMap,
		ring:        ring,
		memberMutex: memberMutex,
		serfEvents:  serfEvents,
		serf:        nodeSerf,
	}
}

// Sets serf metadata.
// See http://godoc.org/github.com/hashicorp/serf/serf#Serf.SetTags for more
// information
func (c *Cluster) SetTags(tags map[string]string) error {
	logger.Info("Setting node tags: %v", tags)
	return c.serf.SetTags(tags)
}

// Starts the Serf protocol and begins listening for Serf events.
func (c *Cluster) Run() {
	logger.Info("Running node")
	c.joinSerfCluster()

	for {
		select {
		case e := <-c.serfEvents:
			c.handleSerfEvent(e)
		case <-c.exit:
			c.exit <- true
			return
		}
	}
}

// Gracefully leaves the Serf cluster and terminates background tasks
func (c *Cluster) Stop() {
	logger.Info("Stopping node")
	c.leaveSerfCluster()
	c.exit <- true
	<-c.exit
}

// Returns a channel of Serf members for a given key.  The first member in the
// channel is "replica 0".  All members of the cluster (including failed nodes)
// will be written to the channel once, then it will be closed.  Nodes which have
// left the clsuter gracefully or have been reaped will not be included.
//
// The first N members in the channel can be seen as a key's "preference set" as
// described in the dynamo paper.
func (c *Cluster) MembersForKey(key string) chan *serf.Member {
	logger.Info("Getting members for key: %s", key)
	return c.ring.membersForKey(key)
}

// Same as MembersForKey, but takes a partition rather than a key
func (c *Cluster) MembersForPartition(partition uint) chan *serf.Member {
	logger.Info("Getting members for partition: %d", partition)
	return c.ring.membersForPartition(partition)
}

func (c *Cluster) joinSerfCluster() {
	if len(c.config.BootstrapHosts) > 0 {
		c.serf.Join(c.config.BootstrapHosts, true)
	}
}

func (c *Cluster) leaveSerfCluster() {
	c.serf.Leave()
}

func (c *Cluster) handleRingChange(event *serf.Event, old_ring *Ring, new_ring *Ring) {
	for partition := uint(0); partition < c.config.Partitions; partition++ {
		old_members := old_ring.membersForPartition(partition)
		new_members := new_ring.membersForPartition(partition)

		if c.config.Releases != nil {
			for replica := uint(0); replica < c.config.Redundancy; replica++ {

				// If partition/replica used to be owned by the local node
				old_member, ok := <-old_members
				if !ok {
					break
				}

				if old_member != nil && old_member.Name == c.serf.LocalMember().Name {
					// ...but isn't any longer
					new_member := new_ring.member(partition, replica)
					if new_member == nil || new_member.Name != c.serf.LocalMember().Name {
						event := ReleaseEvent{
							Partition: partition,
							Replica:   replica,
							To:        new_member,
							SerfEvent: event,
						}

						c.config.Releases <- event
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
					old_member := old_ring.member(partition, replica)
					if old_member == nil || old_member.Name != c.serf.LocalMember().Name {
						event := AcquireEvent{
							Partition: partition,
							Replica:   replica,
							From:      old_ring.member(partition, replica),
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
		logger.Debug("Handling member join event")
		go c.addEventMembers(e)

	case serf.EventMemberLeave:
		logger.Debug("Handling graceful member exit event")
		go c.removeEventMembers(e)

	case serf.EventMemberFailed:
		logger.Debug("Handling unresponsive member event")
		go c.updateEventMembers(e)

	case serf.EventMemberUpdate:
		logger.Debug("Handling member metadata update event")
		go c.updateEventMembers(e)

	case serf.EventMemberReap:
		logger.Debug("Handling forced member exit event")
		go c.removeEventMembers(e)

	default:
		logger.Warning("Unhandled Serf event: %#v", e)
	}
}

func (c *Cluster) recomputeRing() {
	keys := make([]string, 0, len(c.memberMap))
	for k, member := range c.memberMap {
		if c.hasWatchedTag(member) {
			keys = append(keys, k)
		}
	}

	members := make([]*serf.Member, len(keys), len(keys))
	if len(keys) == 0 {
		c.ring = &Ring{members: members}
		return
	}

	sort.StringSlice(keys).Sort()

	for i, k := range keys {
		members[i], _ = c.memberMap[k]
	}

	c.ring = &Ring{members: members}
}

func (c *Cluster) hasWatchedTag(member *serf.Member) bool {
	for tag, re := range c.config.WatchTags {
		member_tag, ok := member.Tags[tag]
		if !ok {
			continue
		}
		if !re.MatchString(member_tag) {
			continue
		}

		return true
	}

	return false
}
