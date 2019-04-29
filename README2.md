# Surprisingly Easy Offline-First Applications

"[Offline-First](https://www.google.com/search?q=offline+first)" describes an application architecture where
data is read and written from a local database on user devices, then synchronized lazily with servers whenever
there is connectivity.

These applications are highly desired by product teams and users because they are much more responsive and
reliable than applications that are directly dependent upon servers.

Unfortunately, mobile-first applications have historically been very challenging to build. Bidirectional
sync is a famously difficult problem, and one which has elluded satisfying general
solutions. Existing products (Apple CloudKit, Android Sync, Google FireStore, Realm, PouchDB) all have at
least one or more serious problems, incuding:

* Requiring that developers write code to handle merge conflicts. This is a variant of concurrent programming
and is quite difficult for developers to do correctly, and a large burden on app teams.
* The lack of ACID transactions. A replicated database that offers automatic merging, but not transactions
isn't really that helpful because developers still have to think carefully about what happens when sequences
of operations are interleaved.
* A restrictive or non-standard data model, for example offering only CRDTs.
* Requiring the use of a specific, often proprietary database on the server-side.

For these reasons, these products are often not practical options for application developers, leaving them
forced to develop their own sync protocol at the application layer if they want an offline-first app, an
expensive and technically risky endeavor.

# Introducing Replicant

Replicant makes it dramatically easier to build these "offline-first" mobile applications. So much easier,
we believe, that there is little reason for any mobile developer not to do so.

The key features that contribute to this leap in usability are:

* **Transactions**: Replicant supports complex multikey read/write transactions. Transactions are arbitrary
functions in a standard programming language, and run serially and completely isolated from each other.
* **Conflict-free**: Virtually all conflicts are handled naturally by the protocol. All nodes are guaranteed
to resolve to the same state once all transactions have been synced (aka "[strong eventual consistency](https://en.wikipedia.org/wiki/Eventual_consistency#Strong_eventual_consistency)"). Developers,
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

In Replicant, we turn the knob further. Like in Calvin, Replicant transactions are pure functions in a
fully-featured programming language.

Unlike Calvin, nodes do not coordinate synchronously to establish order,
or for any other reason. Instead nodes execute transactions completely locally, responding immediately to the calling
application. A log is maintained at each node of the local order transactions occurred in. Asynchronously, when
connectivity allows, nodes synchronize these logs to establish a total order for all transactions. This order
is decided authoratively by one logical node, called the "Replicant Server". This log is then replicated to each
other node (called "Client Node" or "clients").

This will commonly result in a client node learning about transactions that occurred "in the past" from its
point of view (because they happened on disconnected node). In that case, the client rewinds its database back to
the point of divergence and replays the transactions in the correct order.

Thus, once all nodes have the same log, they will execute the same sequence of transactions and arrive at the
same database state. What's more, as we will see, most types of what are commonly termed "merge conflicts"
are gracefully handled in this model without any extra work from the application developer.

# Details

## System Architecture

A deployed system of replicant nodes consists of a single logical "Replicant Server" and one or more "Replicant Clients", which are typically mobile apps running in iOS or Android. Traditional desktop apps and web apps could also be supported.

<diagram, argh>

One or more Replicant Servers are run by the Replicant Service. Typically each "Replicant Server" corresponds to a single user or device.

The basic promise of Replicant is that Replicant Clients are *always* kept in sync with their Server. Once all synchronization is complete, the clients and their server are guaranteed to be in the exact same state. There is no way for application code that is using Replicant (at either the client or server layer) to do something that would prevent the databases from eventually converging.

The clients embed Replicant and use it as their local datastore. In the background Replicant continuously synchronizes with the server.

## Server Responsibilities

The server's only required responsibility is to provide a reliable log service that clients can access with the following operations (provided here in Go-like pseudo-code):

```go
type Op struct {
  // Unique ID of the transaction
  // Generated at the client-side and immutable, even across reordering
  // Once a transaction is submitted on a node, it will be in the final shared log
  ID string

  // Unique ID of the function that was invoked. Typically this is the hash of the
  // code of the function, or some other identifier to find the exact code to invoke.
  FuncID string
  
  // The arguments that the operation was invoked with
  Args []interface{}
}

// Ensures that zero or more operations are in the log. If the entries already exist
// in the log (as identified by their ID), nothing happens. Otherwise the entires are
// appended to the log.
// The return value is the slice of the log from the entry after lastKnownHeadID to
// the new head. This will include `ops`, but also any entries from other clients
// since the last time the caller synced.
// Note: if the requirement to de-dupe is overly burdensome, it can be removed at
// the expense of some additional work client-side.
// Note: the implementation doesn't need to be atomic.
Sync(lastKnownHeadID string, newOps []Op) []Op
```

***TODO:** Is the requirement to not duplicate entries a major complexity for the server? Duplicates could be allowed, it just moves additional complexity to the clients.*

## Client State

A replicant instance maintains the following persistent state:

* Some versioned, forkable database that stores the actual application state
* An ordered log of transactions that determine the current state of the local database
* For each entry in the log, a pointer to the database state at that moment in time
* A pointer to the last known head of the remote database

For the actual embedded database, we use [Noms](https://github.com/attic-labs/noms), a versioned, forkable, transactionable database with efficient one-way replication. But any database could be used as long as it supports atomic transactions, efficient snapshots, and a way to fork from a historical snapshot.

## Data Model

The data model exposed to user code is a fairly standard document database approach.

- keys are byte arrays
- values are JSON-like trees, except:
  - special _class field supported to give json objects a "type", which type that they can later be queried by
  - special _id field for unqiue id
  - blobs supported
- you can query into a subtree of a value using a path syntax
- you can optionally declare indexes on any path

This probably needs more work. I haven't thought a lot about it because it's not relevant to the core problem Replicant is solving, only the developer ergonomics (which is also important! but can be done a bit later).

## Transactions

Interaction with the Replicant database is via _transactions_ which are arbitrary pure functions in some standard programming language.

The language choice is still under investigation. The key desiredata:

* *Determinism*: Every invocation with the same database state and parameters must result in the same output
and effect on the database, on all platforms Replicant runs on.
* *Popularity*: Replicant cannot be easy to use if it requires you to learn a new programming language. Also
popularity on each target platform needs to be consider. For example, Matlab is popular, but it's not popular
with Android or iOS developers.

I am currently thinking that the initial transaction language should be JavaScript. Determinism would be enforced
either using an apporach like [deterministic.js](https://deterministic.js.org/) or by running a JavaScript
interpreter inside [wasmi](https://github.com/paritytech/wasmi) or maybe a forked [Otto](https://github.com/robertkrimen/otto) that enforced determinism. Research should be done into the performance of various approaches.

A second, later language choice could be Rust (on top of wasmi). This is a popular choice in the blockchain space,
where they also require this property of determinism.

## Registering Transactions

Client code *registers* transaction types by some unique identifier (typically a hash) with Replicant. The registrations are stored in-memory.

It might look something like this (from Java):

```java
replicant.RegisterTransactions("transactions.js")
```

And `transactions.js` would be some embedded resource in the Android application containing the various available transactions:

```js
createUser(name, email) {
  if (db.find({
    _class: 'User',
    email,
  }) {
    throw new Error(`User with email %s already exists`, email);
  }

  return db.put({
    name,
    email,
    _class: 'User',
  });
}

createGame(userIDs) {
  const game = db.put({
    userIDs,
    _class: 'Game',
  });

  for (uid of userIDs) {
    const user = db.get(uid);
    user.currentGame = game._id;
  }
  
  return game;
}

updateHighScore(userId, score) {
  const user = db.get(userId);
  user.highScore = Math.max(user.highScore, score);
  db.set(user);
}
```

## Executing Transactions

Client code invokes transactions by hash, or more likely by name for convenience:

```
replicant.exec("updateHighScore", user.ID, newScore);
```

The transaction is run against the current Noms database resulting in a new database state. The log is atomically updated appending the new transaction and parameters.

## Synchronization

Synchronization is a two-step process that should feel reminiscent to anyone who has used git:

1. Push:
  - Replicant sends a list of all ops that are new since the last known server op
  - The result of Push() is a sequence of ops that need to be applied to the last known server op. This might just be the same ops replicant just sent, or it might include ops from other clients.
  - In the case where the list of ops is unchanged, the push is a *fast-forward*. In that case, just set the last-known server op to the last op that was sent to the server and exit.
  - Otherwise:
    - Set a new in-memory pointer `rebaseHead` to the last-known head of the remote log
    - For each op in the returned list from `Push`:
      - Re-run that op atop the `rebaseHead`
      - Set `rebaseHead` to the resulting state
    - Set the last known server head to `rebaseHead`
 2. Rebase:
   - Rebase any new ops from the local log that aren't present in the server log (e.g., ops that occurred since Push() was invoked) in the same way as above

## Conflicts

There are a lot of different things that people mean when they say "conflicts". Let's go through some of them:

### A single read-write register

## Versioning Transactions

# Future Work

## Out-of-Protocol Writes

## Privacy: Server-Proofing the Log Service

## Optimizations
- local (parallelism via deterministic locks, ala calvin)
- remote (hinting of affected keys)
- running Noms on the server

## P2P Finalization

## Edge Database
