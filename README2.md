# Delightfully Easy Offline-First Applications

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
very similar to FireStore, Mongo, Couchbase, Fauna, etc. You don't need to learn anything new, and can build
arbitrarily complex data structures on this primitive that are still conflict-free. You don't need a special `Counter` datatype to model a counter. You just use arithmetic.
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
application. A log is maintained at each node of the order transactions occurred in. Asynchronously, when
connectivity allows, nodes coordinate with an external service (typically the application's own servers)
to establish a total order for all transactions across all nodes. This log is then replicated to each node.
This will commonly result in a node learning about transactions that occurred "in the past" from its point
of view (because they happened on disconnected node). In that case, the node rewinds back to the most recent
shared state and replays the transactions in the correct order.

Thus, once all nodes have the same log, they will execute the same sequence of transactions and arrive at the
same database state. What's more, as we will see, most types of what are commonly termed "merge conflicts"
are naturally handled in this model without any extra work from the application developer.

# Details

## Life of a Transaction

1. Application code in Node N1 registers transactions T1 and T2 with Replicant
  - Transactions can be written in any language, with the restriction that they are pure and deterministic
  - Transactions are identified in Replicant by a hash of their code
  - All transactions get passed at least one parameter, which is the database
  - Transactions are typically written such that they have various preconditions and invariants that they enforce,
    e.g., valid numeric ranges, global uniqueness constraints, foreign key constraints, etc.
2. Application code in Node N1 executes T1 and T2. Both transactions succeed.
3. Replicant instance R1 inside N1 appends to its local log the invocation of T1 and T2.
4. Application code in Node N2 registers the same transactions T1 and T2.
5. Application code in Node N2 executes T1 and T2.
6. Application code in N1 issues a sync request with the server
  - The request includes the transaction which was the latest transaction in the log the last time N1 synced with the server
  - The request includes all novel transactions that N1 has executed since it last synced with the server
7. The server receives the sync request and appends any novel invocations to its log
  - Note that this means that causal consistency is maintained since transactions will always follow transactions they depended on.

... todo ...

## Database

The design of Replicant requires a handful of key features from whatever underlying database it uses:

* Efficient snapshots, because we need to rewind to shared states commonly
* Forking, not just one linear history, because during sync, we want to integrate changes from the server on a branch so that local history can continue to progress during sync
* Determinism, if two nodes start at the same state and run the same sequence of transactions, they must arrive at the same state

Although many databases could theoretically be used or made to work, [Noms](https://github.com/attic-labs/noms) is perfectly suited for this application without any changes.

Additionally, Noms has a few other really useful features for us:

* It is hash-based, so determinism can be trivially verified at all times
* It has efficient one-way replication - you don't need to replay transactions for one way replication, you can just sync the data directly, which is much faster, especially when adding a new node to a group
* It is written in Go, which can be compiled to native code for use on either iOS or Android
* It's quite fast, with peformance comparable to top key/value stores for many workloads
* It has built-in support for indexes to support queries

## Transactions

Interaction with the Replicant database is via _transactions_ which are arbitrary pure functions in some standard
programming language.

The language choice is still under investigation. The key desiredata:

* *Determinism*: Every invocation with the same database state and parameters must result in the same output
and effect on the database, which means the code must follow the same execution. This is a surprisingly uncommon
feature in languages.
* *Popularity*: Replicant cannot be easy to use if it requires you to learn a new programming language. Also
popularity on each target platform needs to be consider. For example, Matlab is popular, but it's not popular
with Android or iOS developers.

I am currently thinking that the initial transaction language should be JavaScript. Determinism would be enforce
either using an apporach like [deterministic.js](https://deterministic.js.org/) or by running a JavaScript
interpreter inside [wasmi](https://github.com/paritytech/wasmi). Research should be done into the performance of
both though.

A second, later language choice could be Rust (on top of wasmi). This is a popular choice in the blockchain space,
where they also require this property of determinism.

## Data Model

The data model will be:

* key/value pairs
  - keys are byte arrays
  - values are JSON-like trees, except:
    - special _class field supported to give json objects a "type", which type that they can later be queried by
    - special _id field for unqiue id
    - blobs supported
  - you can query into a subtree of a value using a path syntax
  - you can optionally declare indexes on any path

## Conflicts

# Future Work

## Optimizations

## P2P Finalization

## Edge Database
