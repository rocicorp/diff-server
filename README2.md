# Replicant

Delightfully Easy Offline-First Applications

## What's this then?

Replicant makes it **insanely easy** to build high-quality "offline-first" mobile applications.

For the first time ever, it is possible to create offline-first mobile apps:

* Without writing conflict resolution code
* With a simple, standard data model (Replicant is a "Document Database" where entries are key/value pairs and the values are JSON objects)
* With real ACID-compliant transactions
* Without depending on a custom, proprietary backend

## Intuition

Replicant is heavily inspired by [Calvin](http://cs.yale.edu/homes/thomson/publications/calvin-sigmod12.pdf).
The key insight in Calvin is that the problem of ordering transactions can be separated from the problem of
executing transactions. As long as transactions are pure functions, and all nodes agree to an ordering, and
the database is a deterministic, then execution can be performed coordination-free by each node independently.

This insight is used by Calvin to create a high-throughput, strictly serialized CP database without the need
for physical clocks. Calvin nodes coordinate synchronously only to establish transaction order, then run their
transactions locally.

In Replicant we turn the knob further: nodes do not coordinate synchronously to establish order, or for any
other reason. Instead nodes rely on an external service (typically the application's own server) to establish
a total order for all transactions. Transactions are executed locally on each node. During synchronization,
nodes post their novel log entries to the server, creating a total order that is compatible with the partial
ordering that actually happened. The merged log is then copied back to the node. Transactions that happened
on other nodes will typically be received out of order. When this happens, the state of the database is
rewound to the most recent shared state, and the transactions are then replayed in the correct order.

As a result, all nodes are guaranteed to arrive at the same state. And since transactions are serialized,
merging the result of parallel operations becomes integrated into the transaction itself and becomes
essentially automatic for almost all cases.

## Details

Developers interact with a Replicant database via *transactions*. A transaction in Replicant is a pure function,
written in some standard programming language (let's assume Go but could be anything). The environment
transactions are run in is controlled to ensure that there are no non-deterministic inputs.

Each transaction receives a number of parameters along with the database. All a transaction can do is return
data to the caller or write to the database. The state of the database at the end of the transaction is
completely determined by the state of the database prior to the transaction, and the parameters passed in.

Therefore, the state of a Replicant database at any point in completely determined by the history of transactions
that have been executed against it. Two devices that start with a Replicant database in a shared state and
execute the same sequence of transactions are guaranteed to end at the same state.
    
