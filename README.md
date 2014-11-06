# libring - distributed hash ring for Go

`libring` is a tool for distributing a set of keys across a cluster of
nodes and routing requests to the node responsible for a given key.
Cluster membership is based on Serf's gossip protocol, and keys are partitioned
across nodes using a type of consistent hashing which minimizes data transfer
when nodes enter or leave the cluster.  Cluster topology changes are exposed as
a channel of acquire/release events.

THIS HAS NOT BEEN TESTED AT SCALE!  Serf is a mature and well-tested system,
however none of the rest of this codebase has been used in large-scale
production systems.  Caveat emptor.

[API Docs](http://godoc.org/github.com/kerinin/libring)

## Design

### Membership

`libring` uses [Serf](https://www.serfdom.io/) for membership.  Serf is "a
decentralized solution for cluster membership, failure detection, and
orchestration".  Serf uses a p2p gossip protocol to maintain an
eventually-consistent view of a set of nodes in a cluster.  

Serf nodes can be tagged with arbitrary key/value pairs, and `libring` uses this
metadata as a way to filter nodes.  The current state of all cluster members is
kept in memory, but only members with a matching tag are used for resolving keys
to members.

`libring` can join Serf clusters running the serf agent (ie running serf but not
libring).  This can be useful in cases where `libring` is used as a routing
layer but doesn't run on the hosts being routed to.  


### Key distribution

`libring` uses the hashing algorithm described as 'Strategy 3' in the appendix
of the [Amazon Dynamo
Paper](http://www.allthingsdistributed.com/files/amazon-dynamo-sosp2007.pdf).
The keyspace is statically mapped to a fixed number of partitions.  Partitions
are assigned to nodes using a consistent hashing algorithm that minimizes
partition relocations under membership changes.  Replication is accomplished by
picking N adjacent nodes along the 'ring' for each partition.

Keys are mapped to partitions by hashing the key name using the FNV32a hash
algorithm, then modding by the partition count to produce an integer partition
identifier.

Partitions are mapped to nodes using [Google's "jump" consistent hash
algorithm](http://arxiv.org/abs/1406.2294).  Node names are sorted alphabetically,
and the resulting array is indexed into using the hashed partition identifier. 
Replicas are assigned by incrementing the index by the replica index (0-N).  By
'walking' the ring of nodes, a 'preference set' can be generated as described in
the Dynamo paper (this can be useful for implementing 'sloppy-quorum' with
hinted handoff, for example).


## Use

First, configure the cluster

```go
config := libring.Config{
  // Specify a set of tag/values which must be present on a Serf member to be
  // treated as part of the cluster.  Allows multiple clusters to share members,
  // and allows members to communicate about their current state
  WatchTags: map[string]*regexp.Regexp{"ring": regexp.MustCompile(`active`)},
  
  // Join the Serf cluster that these hosts are part of.  Can be pointed at a 
  // load balancer if you hostnames are dynamically assigned.
  BootstrapHosts: []string{"loadbalancer.service.com"},

  // Specifies the hostname to use for the local node.  Defaults to the
  // OS-provided value
  HostName: "my_custom_hostname",

  // The number of partitions to divide the keyspace into.  This value should be
  // an order of maginitude larger than the number of members you expect to
  // participate in the cluster.  Larger values increase the amount of metadata
  // the cluster must track, but smaller values limit the scalability of the
  // cluster.  The partition count is an upper-bound on the number of hosts
  // which can participate in a cluster
  Partitions: 2056,

  // Partitions will be assigned to this many hosts at any given point in time.
  // This represents a lower bound on the number of hosts you should have
  // running at any point in time.
  Redundancy: 2,

  // The serf client will be created with this configuration, so if you need to 
  // do anything unusual you can set it here.  Note that libring will specify
  // the EventCh, specifying it in this config is an error.  (If you need to
  // handle raw serf events, you can provide a channel to SerfEvents below)
  SerfConfig: serf.DefaultConfig(),

  // If provided, serf events will be pushed to this channel *after* they have
  // been processed by libring.  Note that the same warning applies here as
  // to serf.Config.EventCh: "Care must be taken that this channel doesn't 
  // block, either by processing the events quick enough or buffering the 
  // channel"
  SerfEvents make(chan serf.Event),

  // Channels for receiving notifications when partitions are assigned to the
  // local machine or removed from the local machine.  Events contain the partition 
  // identifier, the 'other' Member, and the serf Event which triggered the 
  // partition to be reassigned.
  Acquisitions: make(chan AcquireEvent),
  Releases make(chan ReleaseEvent),
}
```

Now you can create a cluster and run it.

```go
cluster := NewCluster(config)

go func() {
  for acquisition := range config.Acquisitions {
    // Do whatever needs to be done in here
    if acquisition.From != nil {
      fmt.Sprintf("Acquired partition %d, replica %d from %s", acquisition.Partition, acquisition.Replica, acquisition.From.Name)
    }
  }
}()

go func() {
  for release := range config.Releases {
    // Do whatever needs to be done in here
    if release.To != nil {
      fmt.Sprintf("Release partition %d, replica %d to %s", acquisition.Partition, acquisition.Replica, acquisition.To.Name)
    }
  }
}()

// If this host should be part of the cluster, update its tags.
cluster.SetTags(map[string]string{"ring": "1"})

go cluster.Run()
```

This will fire up Serf and start talking to the other machines in the cluster.
Now you can use your shiny new cluster to route requests to nodes

```go
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
  
  path := r.URL.Path
  nodeForPath := <-cluster.MembersForKey(path)
  
  fmt.Printf("Proxying %s to node %v", path, nodeForPath)

  proxy := httputil.NewSingleHostReverseProxy(nodeForPath.URL)
  proxy.ServeHTTP(w, r)
})

log.Fatal(http.ListenAndServe(":8080", nil))
```
