# Welcome to Replicant.

Replicant makes it easy - pleasant, even - to create insanely fast, local-first, offline-enabled mobile applications. All UI interactions are local by default, whether or not there is connectivity. The database is synchronized lazily with the server, and there is virtually no manual conflict resolution by either users or developers.

TODO: Diagram

## How it Works

Replicant provides an embedded database that is accessible on both the client and server-side of your application. Think of it like SQLite, but where you can also directly modify the SQLite instance on the server-side.

Any change you make on the client-side shows up on the server the next time the client synchronizes, and vice-versa.

Because the nodes commuicate asynchronously, conflicts can occur if the same data is modified concurrently at both ends. Such conflicts are handled in Replicant naturally and intuitively by expressing mutation as atomic transactions, written in JavaScript. Replicant rewinds and replays these transactions as necessary on each node, so that all nodes end up seeing the same sequence of changes. Since the transactions are pure functions, this guarantees all nodes end in the same state.

This ends up being a really nice way to work with client/server applications: in most cases you can completely ignore conflicts, and those that remain are far easier to reason about.

Applications can create as many Replicant instances as they need, but a typical pattern is one-per-user. On the server side, the application puts any data into the instance that the application will need locally/offline for that user. On the client side, the application uses Replicant as its local datastore, reading and writing to it in place of something like SQLite. The application then monitors the server side of the instance for changes coming from the client and reflects them into the rest of the system.

For more detail, see the full design doc.

# Get Started

* Quickstarts:
  * [Flutter](./doc/quickstart-flutter.md)
  * [React Native](./bind/react-native/README.md)
  * iOS/Swift (soon)
  * iOS/Objective-C (soon)
  * Android/Kotlin (soon)
  * Android/Java (soon)
* [JavaScript Transaction API](./doc/transaction-api.md)
* [HTTP API](./doc/http.md)
* [`repl` CLI](./doc/cli.md)
* [Authentication/Authorization](./doc/auth.md)
* [Design Document](./doc/design.md)
