# Delightfully Easy Offline-First Applications

## Problem

"Offline-First" describes a mobile application architecture where data is read and written from a local
database on the device, and synchronized lazily with the server when there is connectivity.

These applications are highly desired by product teams and users because they are so much more responsive,
reliable, and resilient to variable network conditions.

Unfortunately, mobile-first applications have historically been very challenging to build. Bidirectional
sync is a famously difficult problem in computer science, and one which has elluded satisfying general
solutions. Existing products (Apple CloudKit, Android Sync, Google FireStore, Realm, PouchDB) all have at
least one or more serious problems, incuding:

* Requiring developers write code to handle merge conflicts. This is a variant of concurrent programming
and is quite difficult for developers to do correctly, and a large increase in application complexity.
* The lack of ACID transactions. A replicated database that offers automatic merging, but not transactions
isn't really that helpful. Because it still forces the developer to think about what happens when multistep
operations interleave.
* A restrictive or non-standard data model.
* Requiring the use of a specific, often proprietary database on the server-side.

For these reasons, these products are often not practical options for application developers, leaving them
forced to develop their own sync protocol at the application layer if they want an offline-first app, an
incredibly expensive and technically risky endeavor.

# Introducing Replicant

Replicant makes it dramatically easier to build high-quality "offline-first" mobile applications.

The key features that make Replicant so easy to use are:

* **Transactional**: Replicant supports complex multikey read/write transactions. Transactions are run
serially and completely isolated from each other. Transactions either succeed or fail atomically.
* **Conflict-free**: Virtually all conflicts are handled naturally by the protocol. All nodes are guaranteed
to resolve to the same state once all transactions have been synced ("strong eventual consistency"). Developers,
in almost all cases, do not need to think about the fact that nodes are disconnected. They simply use the database as if
it was a local database and synchronization happens behind the scenes.
* **Standard Data Model**: The replicant data model is a simple document database. From an API perspective, it's
very similar to FireStore, Mongo, Couchbase, etc. You can build arbitrarily complex datamodels that maintain their
correctness guarantees. You don't need a special `Counter` datatype to model a counter. You just use plain arithmetic.
* **Open**: Replicant has extremely minimal requirements on the server-side. It can work with any existing
server-side stack.

# Intuition

Replicant is heavily inspired by [Calvin](http://cs.yale.edu/homes/thomson/publications/calvin-sigmod12.pdf).
The key insight in Calvin is that the problem of ordering transactions can be separated from the problem of
executing transactions. As long as transactions are pure functions, and all nodes agree to an ordering, and
the database is a deterministic, then execution can be performed coordination-free by each node independently.

This insight is used by Calvin to create a high-throughput, strictly serialized CP database without the need
for physical clocks. Calvin nodes coordinate synchronously only to establish transaction order, then run their
transactions locally.

In Replicant we turn the knob further: nodes do not coordinate synchronously to establish order, or for any
other reason. Instead nodes execute transactions completely locally, responding immediately to the
application. A log is maintained at each node of the order transactions occurred in. Asynchronously, nodes
coordinate with an external service (typically the application's own servers) to establish a total order
for all transactions across all nodes. This log is then replicated to each node. This will commonly result
in a node learning about transactions that occurred "in the past" from its point of view (because they
happened on disconnected node). In that case, the node rewinds back to the most recent shared state and
replays the transactions in the correct order.

Thus, once all nodes have the same log, they will execute the same set of transactions and arrive at the
same database state. What's more, as we will see, most types of what are commonly termed "merge conflicts"
are naturally handled in this model without any extra work from the application developer.

# Details

## Transactions

## Data Model

## Deterministic Database

## Conflicts
