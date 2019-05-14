# API Design Thoughts

Replicant is an interesting piece of software in that it needs to have bindings to so many places:

* iOS (Objective-C and Swift)
* Android (Java and Kotlin)
* JavaScript (from inside transaction functions)
* The Web (if there is to be a web implementation)
* CLI (for debugging, exploration)
* HTTP (for sync and server-side integration)

Longer-term, there is also:

* Windows/OSX (for electron/desktop app support)
* Dart
* ... etc ...

This is obviously *way* too much to support if each bindings layer was done individually. We need some
kind of narrow waist above which all these can be reflected more or less automatically.

## Plan

At a low level, the interface to replicant is exposed in terms of "commands" (see the cmd package).
These are analgous to a CLI, but phrased in terms of Go structs, not strings.

The cmd interface is reflected automatically into several places:

* A C API
  - Consumed by most client embeddings directly: iOS, Android, Windows, OSX, etc.
  - Will likely have libraries on top provided by users that make it easier to use, but those are "out of scope", at least initially, for Replicant itself.
* The CLI
* JavaScript bindings
* HTTP

This design doesn't result in the most beautiful interface for any of these targets, but it saves Replicant
developers from drowning in bindings work.

## Performance

During the development of Noms, we found that the most overall winning perf strategy was to avoid allocations.
Allocations are themselves expensive, but worse stress GC, which is the biggest perf killer. This is why Noms
structs are "lazy decoded" for example. When you use Noms via the Go API, it is typically zero-copy.

In order to get a usable API quickly, Replicant opts to expose the first interface as JSON. This implies at
least one copy for every read and write, to translate from JSON to Noms and back.

Eventually, we could, and should, directly expose the Noms type system directly to clients. It can *feel* very
JSON-like, so ergonomics are maintained. People could still start out with the JSON interface to get going, and
graduate to real Noms as their needs develop.

This would imply both much more sophistiicated bindings layer on the Go side, and also richer bindings on the
host side.

## Performance (2)

Another thought on performance:

Currently `replicant put` is converting JSON to Noms. We could not do that. We could just store the JSON, in
a Noms Blob type. It is certainly possible to read JSON in a streaming way, so even subpath reads could be
supported. The main downside of this strategy is that the rest of the existing Noms tools use a lot of their
utility (e.g., CLI tools like `noms insert` could not be used as-is).
