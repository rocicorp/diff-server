# Welcome to Replicant.

Replicant makes it easy - pleasant, even - to create insanely fast, local-first, offline-enabled mobile applications.

Replicant works along side your existing stack. It provides a simple primitive: an asynchronous bidirectional pipe that is accessible on both the client and server sides of your application.

TODO: Diagram

Any data you put on one side of the pipe shows up at the other side the next time the pipe is synchronized. Since the pipe is asynchronous, conflicts can occur if the same data is modified concurrently at both ends. Such conflicts are handled naturally and intuitively by expressing mutation as atomic transactions, written in JavaScript. Replicant rewinds and replays these transactions as necessary so that both sides of the pipe end up seeing the same sequence of changes. Since the transactions are pure functions, this guarantees that both ends of the pipe converge to the same state.

This ends up being a really nice way to work with disconnected applications: in most cases you can completely ignore conflicts, and those that remain are far easier to reason about.

Applications can create as many pipes as they need, but a typical pattern is one-per-user. On the server side, the application puts any data into the pipe that the application will need locally/offline for that user. On the client side, the application uses Replicant as its local state, reading from and writing to it, in place of soemthing like SQLite. The application then monitors the server side of the pipe for changes coming from the client and reflects them into the rest of the system.

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
* [Design Document](./doc/design.md)
