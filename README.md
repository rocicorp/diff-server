# Summary

Applications frequently need to store data client-side to support disconnected operation.

Such applications require some form of _replication_ in order to periodically:

1. Push data to the server and peers
2. Get new data from the server and peers

Existing databases used in this model (PouchDB, Firebase, Realm, Noms, etc) have one or more of the following limitations:

* Developers must manually handle at least some types of conflicts. This can be quite complex to do correctly.
* Lack of support for multikey/multistatement transactions. In a concurrent setting, implementing correct applications without
  transactions is very difficult.
  
Replicant is a consensus layer that can be added on top of any existing database that turns that database into an
automatically-synchronizing transactional database without requiring the devleoper to manually merge concurrent changes
or handle conflicts.

Formally, the resulting database supports sticky availability and casual+ consistency.

Manual conflict resolution is not required. The tradeoff is that some transactions that one peer runs and
sees succeed locally may later be rolled back if conflicting concurrent transactions occurred. However, note that the
same is true in some sense for any totally or sticky available database, since writes one peer performs might be undone
by conflicting writes on another peer.

# Intuition

Replicant draws inspirating from Calvin/FaunaDB. The key insight in Calvin is that the problem of _ordering_
transactions can be separated from the problem of _executing_ transactions. As long as transactions are pure functions,
and all nodes agree to an ordering, then execution can be performed coordination-free by each node independently.

This insight is used by Calvin/FaunaDB to create a strictly serialized CP database. Nodes coordinate only to establish
transaction order, then run the transactions locally.

In Replicant we turn the knob further: Nodes do not coordinate to establish order, or for any other reason. Instead of coordinating to establish global order, nodes calculate a global order for transactions locally, using a deterministic function of the transaction, its parent, and its parameters (e.g., a hash).

Transactions are gossiped between nodes asynchronously, whenever they are able to connect. Transactions will be received
out of order. When this happens, the state of the database is rewound as necessary, and the transactions are replayed in the correct order.

Consensus is achieved when all (or at least a majority) of known nodes have acknowledged up to a particular point in a
history of transactions.

The result is a disconnected DB with casual+ consistency that allows full multikey/multistatement transactions, and no
manual conflict resolution required of developers.

# Requirements

You will need:

1. Some underlying database
2. A way to efficiently snapshot and restore versions of that database (it's better if the underlying db supports this natively, but not strictly required)
3. A way to serialize multikey/statement transactions and parameters to them

# Sketch

This is clearly non-optimal, just getting the point across:

1. SQLite is used as the underlying datastore
2. Snapshots are accomplished via just copying the sqlite data directory (cow filesystem would be useful here, or even better a database that inherently supports timetravel)
3. Transactions are implemented as JavaScript functions that run in an isolated environment. These functions can access SQLite, but nothing else except for params passed into them.
These functions are pure, they have no access to the outside world. They are also deterministic (clock must be controlled, etc).

## Details

The database is a log of all transactions ever run. Each transaction contains:

* The previous (or _parent_) transaction
* A unique node ID (the node the transaction was originally executed on)
* The code of the transaction (or a hash or other identifier that represents that code)
* The parameters to the transaction
* The list of nodes that have acknowledged the transaction

Because the database is distributed among many nodes, the log can have branches where multiple nodes executed transactions
against a state in parallel.

The state of the DB at any point in time is the summation of all the transactions up to that moment. The order transactions
are run in to calculate this state is deterministic. At any point where the log forks, the transactions are run in order of their hash.

Transactions can fail, for a variety of reasons, including enforcing database invariants. It is up to the developer to check
ensure any invariants are validated inside each transaction. However, depending on underlying database, much of this (e.g.,
foreign keys, indexes) will be delegated.

If a transaction fails, it has no effect on subsequent transactions.

## Consensus/Finalization

A transaction can be considered finalized when all the known nodes (which can be stored in the database) have ack'd the transaction.

When a transaction gets replayed (because it received older transactions from some peer), it will end up having a different
parent, thus its hash will change, and thus its a different transaction. So the set of acknowledged peers gets wiped and
replaced with just the node that did the merge.

Note that it is also possible to tweak the consensus rules. You can decide that consensus is achieved when only a majority of
peers have acked the write. To do that just change the ordering rule of transactions to sort such finalized transactions before others at the same branch point.

This might be useful in cases where nodes are client devices which can go offline for long/indefinite periods.

# Other ideas

Besides being used as a client-side database, Replicant could conceivably be used as a traditional distributed database.
In this configuration, each "node" would likely be a set of nodes in one datacenter, each responsible for a subset of the
data. The result would be a highly available, low-latency, distributed database offering casual+ consistency, not strict
serializability. Such a database would be far more performant than strict serializable databases, and not much harder to
use.

To increase performance further, some concurrent writes could be re-introduced. For example, you could allow two transactions that access different subsets of keys to be run "in parallel". This would mean that such transactions would not need to be
replayed in serial, you'd just merge their effects together.
