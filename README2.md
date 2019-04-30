# Replicant: Surprisingly Simple Offline-First Applications

"[Offline-First](https://www.google.com/search?q=offline+first)" describes an application architecture where
data is read and written from a local database on user devices, then synchronized lazily with servers whenever
there is connectivity.

These applications are highly desired by product teams and users because they are much more responsive and
reliable than applications that are directly dependent upon servers.

Unfortunately, offline-first applications have historically been very challenging to build. Bidirectional
sync is a famously difficult problem, and one which has elluded satisfying general
solutions. Existing attempts to build such general solutions (Apple CloudKit, Android Sync, Google FireStore, Realm, PouchDB) all have one or more of the following serious problems:

* **Requiring that developers manually merge conflicting writes.** Consult the [Android Sync](http://www.androiddocs.com/training/cloudsave/conflict-res.html) or [PouchDB](https://pouchdb.com/guides/conflicts.html) docs for a taste of how difficult this is for even simple cases. Then remember that every single pair of operations that can possibly conflict needs to be considered this way, and the resulting conflict resolution code needs to be kept up to date as the application code changes. Developers are also responsible for ensuring the resulting merge is equivalent on all devices, otherwise they end up in a [split-brain](https://en.wikipedia.org/wiki/Split-brain_(computing)) scenario where nodes have different states but don't know that they do.
* **Lack of atomic transactions.** Some solutions claim to automatically resolve conflicts, but lack atomic transactions. Without transactions, developers are put in the position of reasoning about concurrent execution of any possible sequence of database operations in their application. This is analogous to multithreaded programming without locks or any other kind of concurrency control.
* **A restrictive or non-standard data model.** Some solutions achieve automatic conflict resolution with restrictive data models, for example, only allowing CRDTs. However CRDTs are only known for a few relatively simple datatypes. This forces developers to twist their data model to fit into one of the provided CRDTs. For example, Realm has a special [counter](https://realm.io/docs/java/latest/#field-types) type that merges concurrent changes by summing them. But if you want to implement something very similar - a high score in a game - there is no way easy way to do that in Realm because there is no special `MaxNum` type built into Realm.
* **Reliance on a hosted service for the server-side.** Requiring the use of a third-party hosted service is not an option for many security or privacy conscious organizations.
* **Difficult or non-existent incremental integration path with existing services.** Some solutions effectively require a wholesale migration to a new backend database and architecture.

For these reasons, these products are often not practical options for application developers, leaving them
forced to develop their own sync protocol at the application layer if they want an offline-first app, an
expensive and technically risky endeavor.

# Introducing Replicant

Replicant dramatically reduces the difficulty to build these "offline-first" mobile applications.

The key features that contribute to this leap in usability are:

* **Transactions**: Replicant supports complex multikey read/write transactions. Transactions are arbitrary
functions in a standard programming language, and run serially and completely isolated from each other.
* **Conflict-free**: All nodes are guaranteed to resolve to the same state once all transactions have been synced (aka "[strong eventual consistency](https://en.wikipedia.org/wiki/Eventual_consistency#Strong_eventual_consistency)"). Developers,
in almost all cases, do not need to think about the fact that nodes are disconnected. They simply use the database as if
it was a local database and synchronization happens behind the scenes.
* **Standard Data Model**: The replicant data model is a simple document database. From an API perspective, it's
very similar to FireStore, Mongo, Couchbase, Fauna, etc. You don't need to learn anything new, and can build
arbitrarily complex data structures on this primitive that are still conflict-free. You don't need a special `Counter` datatype to model a counter. You just use arithmetic.
* **Open**: Replicant is designed to integrate incrementally into large existing systems, not insist that you rewrite everything to use it.

# Intuition

Replicant is heavily inspired by [Calvin](http://cs.yale.edu/homes/thomson/publications/calvin-sigmod12.pdf).
The key insight in Calvin is that the problem of ordering transactions can be separated from the problem of
executing transactions. As long as transactions are pure functions, and all nodes agree to an ordering, and
the database is a deterministic, then execution can be performed coordination-free by each node independently.

This insight is used by Calvin to create a high-throughput, strictly serialized distributed database without the need
for physical clocks. Calvin nodes coordinate synchronously only to establish transaction order, then run their
transactions locally.

In Replicant, we turn the knob further. As in Calvin, Replicant transactions are pure functions in a
fully-featured programming language.

Unlike Calvin, nodes do not coordinate synchronously to establish order,
or for any other reason. Instead nodes execute transactions completely locally, responding immediately to the calling
application. A log is maintained at each node of the local order transactions occurred in. Asynchronously, when
connectivity allows, "client nodes" (those running the user interface) synchronize their logs with a special (logical)
node called the "Replicant Server", which decides authoratively what the total order is. The resulting totally ordered log is then replicated back to each client node.

This will commonly result in a client node learning about transactions that occurred "in the past" from its
point of view (because they happened on disconnected node) after synchronizing with the server. In that case,
the client rewinds its database back to the point of divergence and replays the transactions in the correct order.

Thus, once all nodes have the same log, they will execute the same sequence of transactions and are guaranteed to arrive at the
same database state. What's more, as we will see, most types of what are commonly termed "merge conflicts"
are gracefully handled in this model without any extra work from the application developer.

# Data Model

The Replicant data model in inspired by Git and similar systems.

Each change to the system is represented by a `Commit` that points to the complete state of the database as of the commit. The prior change is 

# System Architecture

A deployed system of replicant nodes is called a *Replicant Group* and consists of a single logical *Replicant Server* and one or more *Replicant Clients*. Replicant Clients are typically mobile apps running in iOS or Android, but traditional desktop apps and web apps could also be clients, or really any software that embeds the Replicant Client library.

<p align="center">
  <img src="./replicant.svg" alt="System Architecture Diagram">
</p>

Typically each Replicant Group models data for a single user of a service across all the user's devices. But a Replicant Group could be more fine-grained (if, for example, it's desirable to replicate a different subset of data to different device types) or more coarse-grained (if there are groups of users collaborating on the same dataset).

One or more Replicant Servers are run by the Replicant Service. The Replicant Service is run alongside the application's existing server stack and database of record. Plumbing is added to route relevant updates from the database of record to Replicant Servers and the reverse (see integration).

The key promise of Replicant is that Replicant Clients are *always* kept in sync with their Server. Once all nodes in a group have exchanged all transactions, they are guaranteed to be in the exact same state. There is no way for application code at either the client or server layer to do something that would prevent that from occurring.

This is a powerful promise that makes reasoning about synchronization much simpler.

# Replicant Client

<img src="./replicant-client.svg" alt="System Architecture Diagram" align="right">

A Replicant Client is embedded within a client-side application, typically a mobile app in iOS or Android, but also potentially a desktop or web app. The application, or _host_, uses the client as its local datastore.

The client is updated by executing _transactions_, which are invocations of pure functions called _transaction functions_. Each _transaction function_ takes one or more parameters, plus a snapshot of the current state of the database, and returns as a result a new state of the database.

Theoretically, Replicant could be built atop any single-node database that has the following features:

* transactions - ACID-compliant transactions
* snapshots - previous versions can be kept efficiently
* forking - you can fork the database from any previous snapshot efficiently

However [Noms](https://github.com/attic-labs/noms) - a prior project of ours - is especially well-suited because it has all these features, plus others that will be used by later sections of this document.

You do not need to understand all the details of Noms to understand this document. What you need to understand is that Noms is a versioned, transactional, forkable database. Think SQLite+Git.

## Client State

Replicant maintains two Noms _datasets_ (analagous to Git branches):

* _remote_ - the last-known state of the Replicant Server
* _local_ - the current state exposed to the host application

Each dataset's latest commit has the following Noms type:

```
Struct Commit {
  meta: Struct Meta {
    date: Struct Date {
      NanosSinceEpoch: Number,
    },
    tx: Struct {
      args: List<Value>,
      code: Ref<Blob>,
      name: String,
      origin: String,
    },
  },
  parents: Set<Ref<Cycle<Commit>>>,
  value: Struct {
    txCode: Ref<Set<Blob>>,
    data: Map<String, Value>,
  },
}
```

Each Noms `Commit` represents a transaction in Replicant. The `meta.tx` field describes the transaction that was run that resulted in the commit. Specifically:

* `origin`: The node the transaction was originally run on (useful for debug purposes)
* `code`: The code that contains the transaction function that was invoked (see "registering transactions")
* `name`: The name of the transaction function from `code` that was run
* `args`: The arguments that were passed to the transaction function

The standard `meta.date` field is also used for the current datetime inside the transaction. 

The value of the transaction has two parts:

* `txCode`: All registered transactions (see 'registering transactions')
* `data`: A map of all currently stored user data, by ID (see data model, below)

***TODO:** Indexes need to go here somewhere. They aren't synchronized, but they need to be updated atomically with commits.*

## Data Model

The data model exposed to user code is a fairly standard document database approach, like Google Firestore, Couchbase, RethinkDB, etc:

- keys are byte arrays
- values are JSON-like trees, except:
  - special _class field supported to give json objects a "type", which type that they can later be queried by
  - special _id field for unqiue id
  - blobs supported
- you can query into a subtree of a value using a path syntax
- you can optionally declare indexes on any path

*** TODO:** This needs a lot more work. I haven't thought a lot about it because it's not relevant to the core problem Replicant is solving, only the developer ergonomics (which is also important! but can be done a bit later).*

## Transaction Language

The key desired features for the transaction language are:

* *Determinism*: Every invocation with the same database state and parameters must result in the same output
and effect on the database, on all platforms Replicant runs on.
* *Popularity*: Replicant cannot be easy to use if it requires you to learn a new programming language. Also
popularity on each target platform needs to be considered. For example, Matlab is popular, but it's not popular
with Android or iOS developers.

I am currently thinking that the initial transaction language should be JavaScript. Determinism *could* be **enforced** a variety of ways:

* Using an approach like [deterministic.js](https://deterministic.js.org/) - this is a blacklist approach, and so it's guaranteed to miss things
* Running a JavaScript interpreter inside [wasmi](https://github.com/paritytech/wasmi) - this is a whitelist approach that was built from the ground-up for determinism, but it's slow
* Running inside a forked [Otto](https://github.com/robertkrimen/otto) that enforced determinism - also slow

I think that we do not need determinism to be rock-solid because we will detect non-deterministic transactions automatically during sync. All we need to do is make non-deterministic transactions hard to trigger by accident, and the deterministic.js approach is sufficient for that.

## Invoking Transactions

Since transaction code is stored in the database and synchronized with other data, invoking transactions is simply running the relevant function and writing an appropriate commit to Noms referencing the code.

It might look something like this (in Java):

```java
// Writes the code from "transactions.js.bundle" included in the app to the DB if not present
Transactions txs = replicant.LoadTransactions("transactions.js.bundle");

// Execute "createUser" from the bundle and write the transaction to the database
ReplicantResult result = txs.exec("createUser", newUserName, newUserEmailAddress);
```

However, we expect that in the typical case, applications will want to pre-register transaction code on the server-side for efficiency. See "registering transactions" for more.

# Replicant Server

Structurally, a Replicant Server is very similar to a client. It contains a Noms database and executes transactions in the same way.

However, its role in the system is different: a Replicant Server's main responsibility is to maintain the authorative history of transactions that have occurred for a particular Replicant Group and their results.

Unlike clients, Replicant Servers do not ever rewind. The server is Truth, and the clients dance to its tune. Once a transaction is accepted by a server and written to its history, by either clients or the server itself, it is final, and clients will rewind and replay as necessary to match.

This does not mean, however, that servers have to accept whatever clients write. Servers have full discretion over whether to accept any given transaction, and they validate all work clients do. See "synchronization" for details.

## Noms Schema

The same as the client, except there's only a single dataset, `master`, since the server doesn't need to allow a separate branch to evolve while sync is in progress the way the client does.

## Consistency Requirements

Each Replicant Server acts as a single strictly serialized logical database, even though they are typically a distributed system internally. In the event of a partition internal to the replicant server, it ceases to be available rather than give inconsistent results. Note that this is fine, however, since the clients are designed to be frequently disconnected from their server.

## API

For each Replicant Server, the Replicant Service exposes an API that is the same as the [Noms Remote Server API](https://github.com/attic-labs/noms/blob/master/go/datas/database_server.go#L64), except that `PostRoot()` is non-public and a new `Commit(newHead hash.Hash)` endpoint is added. See Synchronization for details.

## Registering Transactions

We expect that users will typically want to *register* transaction functions at the server-side, rather than let clients execute whatever transactions they want, for a few reasons:

1. Without this, clients would have to include the code in their packages, and then write it into their databases, which would double the amount of storage the clients would consume.
2. We expect that developers will usually want to whitelist transaction functions that can run, based on known hashes of code bundles. Otherwise, malicious clients could attack good clients by way or the sync protocol.
3. For many transaction types originating on clients, there will be server-side actions that need to happen -- either to actually execute the transaction in reality, or to validate the transaction. It's natural to integrate these handlers at the point of registration.

This is implemented as a special pair of transaction functions baked into all Replicant nodes: `registerTransaction` and `unregisterTransaction` that update the `value.txCode` field of the server's `master` dataset. Since these transaction functions could never be themselves registered, they will always fail validation during sync and thus will not be allowed to be called by clients (see Synchronization).

## Replicant Service

The Replicant Service is a stateless, horizontally scalable application server server written in Go that runs one or more Replicant servers. Because Replicant Servers store a small amount of data, there is no need to split the data of a single Replicant Server across multiple servers. However, it may be the case that for a variety of reasons there are multiple instances of the same Replicant Server running at once.

An easy way to meet these requirements is to store all the state in Noms, configured to use S3/Dynamo as its backend (see [NBS-on-AWS](https://github.com/attic-labs/noms/blob/master/go/nbs/NBS-on-AWS.md)). However, one side-effect of doing that naively would be that there would be no data deduplication between Replicant Servers.

# Synchronization

Synchronization is a three-part process that should feel very similar to anyone who has looked under the covers at Git. It takes advantage of Noms' built-in fast one-way replication to accelerate "fast-forward" syncs.

## Step 1: Client pushes to server

The client uses (effectively) [`noms sync`](https://github.com/attic-labs/noms/blob/master/doc/cli-tour.md#noms-sync) to push all missing chunks from the client's `local` dataset to the server's `master` dataset. At the end of the push, the client calls `Commit(newHead hash.Hash)`.

## Step 2: Commit on the server

On the server-side, `Commit(newHead)` looks like:

1. The call is queued behind any other commit to the same Replicant Server. Since Replicant Groups are usually small numbers of nodes, this will typically be a very short wait.
2. When the call continues:
  - Find the fork point between the client's commit and the server's latest commit
  - If the server commit is a fast-forward from client:
    - Respond with the new head, there's nothing more to do
  - Else:
    - Validate each new commit (each commit after the fork point on the client side):
      - Check that the specified transaction codebase is registered (exists in .value.txCode) and the function is known
      - Execute the transaction
      - If the resulting hash doesn't match the one the client specified, the client is badly behaved, return 40x (see badly-behaved clients)
      - If the transaction has server-side validation registered, run that validation (see integration)
        - If the validation fails, replace the transaction with a CommitFailure transaction (see server-side validation)
      - Commit the new head
      - If the client commit is a fast-forward of the validated transaction chain:
        - Return the new head
      - Else:
        - Add a merge commit referencing the two branches and indicating which one goes first (see merge commits)

## Step 3: Client-Side Pull

Back on the client-side, the `Commit()` call has just returned with a new head that should become the head of the `remote` dataset. This is trival. We trust the server and this makes no changes to our local state, so we just `noms sync` the server's `master` dataset to our `remote` dataset, which pulls all the relevant chunks and we're done.

## Step 4: Client-Side Rebase

We want to enable clients to make local progress between Step 1 and Step 3. Otherwise apps will be stalled waiting for syncs that may take awhile, or even stall in the face of flaky networks.

Therefore we allow the `local` dataset to evolve as normal while the sync is in progress.

As a result, after step 3 finishes, we may have some new commits in the `local` dataset since when step 1 started. We must rebase these commits:

- Find fork point between `remote` and `local` heads
- If local is ff of remote (no other client submitted work in meantime)
  - nothing to do
- If remote dataset is ff of local (no local work happened in meantime):
  - Set local to remote
- Else:
  - Rebase each new commit from local fork onto `remote` head
  - Commit result to `local`

# Conflicts

There are a lot of different things that people mean when they say "conflicts". Let's go through some of them:

## A single read-write register based on paramters

Example:

```js
setCellValue(spreadsheetID, row, column, value) {
  db.set(_id: spreadsheetID, `.rows[${row}].cells[${column}]`, value);
}
```

In this example, a transaction takes data from the user and sets a value in the database. If this runs concurrently at two sites, there is no way to merge them. One must win, or we must ask the user.

However, remember that this case isn't just a part of offline applications - it happens in normal client/server apps too. It is perfectly possible for a user to set a cell in their spreadsheet, and then another client overwrites it an instant later. This is not really different.

## Multiple register writes

## Arithmetic

## Accumulation

## Data structure maintenance

## Dependent write

All the above are handled naturally. Then there is:

## Sequence manipulation

Unclear how badly this is needed, but if it is, we can add a Noms type that is a sequence CRDT. Since we know the ancestery of all parallel writes, we can use a CRDT and it will automatically make the correct sequence edits.

# Future Work

## Optimizations
- local (parallelism via deterministic locks, ala calvin)
- remote (hinting of affected keys)
- running Noms on the server

## P2P Database

## Edge Database
