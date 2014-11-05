# libring - distributed hash ring for Go

`libring` is a tool for distributing a set of keys across a cluster of
nodes and routing requests to the node responsible for a given key.
Cluster membership is based on Serf's gossip protocol, and keys are partitioned
across nodes using a type of consistent hashing which minimizes data transfer
when nodes enter or leave the cluster.  Cluster topology changes are exposed as
a channel of acquire/release events.


## Use

First, configure the cluster

```go
config := libring.Config{
  // Specify a set of tag/values which must be present on a Serf member to be
  // treated as part of the cluster.  Allows multiple clusters to share members,
  // and allows members to communicate about their current state
  WatchHosts: map[string]string{"ring": "1"},
  
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
  Redundancy: 2

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

for acquisition := range config.Acquisitions {
  // Do whatever needs to be done in here
  node := <-cluster.MembersForPartition(acquisition.Partition)
  fmt.Sprintf("Partition %d is now the leader for partition %d", node, acquisition.Partition)
}

for release := range config.Releases {
  // Do whatever needs to be done in here
  node := <-cluster.MembersForPartition(acquisition.Partition)
  fmt.Sprintf("%v is no longer the leader for partition %d", node, acquisition.Partition)
}

// If this host should be part of the cluster, update its tags.
cluster.SetTags(map[string]string{"ring": "1"})

cluster.Run()
```

This will fire up Serf and start talking to the other machines in the cluster.
Now, maybe you want so use your shiny new cluster

```go
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
  
  path := r.URL.Path
  nodeForPath := <-cluster.MembersForKey(path)
  
  fmt.Fprintf(w, "%v is responsible for path %v", nodeForPath, path)
})

log.Fatal(http.ListenAndServe(":8080", nil))
```
