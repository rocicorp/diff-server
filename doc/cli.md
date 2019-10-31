# The `repl` CLI

The Replicant SDK includes a command-line program called `repl` which you can use to interactively inspect and
manipulate Replicant databases from the terminal.

To install it, copy the binary for your platform [from the latest release](https://github.com/aboodman/replicant/releases/latest) to someplace in your path and run it:

```
cp <sdk>/replicant-darwin-amd64 ~/bin/repl
chmod u+x ~/bin/repl
```

## Interacting with Replicant databases

Example:

```
$ repl --db=/tmp/mydb bundle put <<HERE
function createUser(id, name, favoriteColor) {
  db.put('user/' + id, {
    name: name,
    color: favoriteColor,
  });
}

function getUsersByColor(color) {
  return db.scan({prefix:'user/'})
    .filter(function(kv) { return kv.value.color == color })
    .map(function(kv) { return kv.value });
}
HERE
Replacing unversioned bundle 2eulo8v8rihcjm0e93brv14dopakkder with 2h9fth56vu4n3rrn9prfae2r8dokt4qe

$ repl --db=/tmp/mydb exec createUser `uuidgen` Abby orange
$ repl --db=/tmp/mydb exec createUser `uuidgen` Aaron orange
$ repl --db=/tmp/mydb exec createUser `uuidgen` Sam green

$ repl --db=/tmp/mydb exec getUsersByColor orange
[
  map {
    "color": "orange",
    "name": "Abby",
  },
  map {
    "color": "orange",
    "name": "Aaron",
  },
]
```

See `repl --help` for complete documentation.

## Interacting with Remote Replicant instances

You can use the CLI to talk to remote replicants to. For example, if you have a server at https://replicate.to/serve/example, you could do:

```
$ echo 42 | repl --db=https://replicate.to/serve/example put foo
$ repl --db=https://replicate.to/serve/example get foo
42
```

## Running a Development Server

You can run a fully-functioning Replicant server against local disk using `repl`:

```
repl --db=~/replicant-storage serve
```

You can then point your Replicant clients (the CLI, using the `--db` flag, as well as the bindings) at http://localhost:7001/serve/sandbox/foo, where `foo` is a unique database name that you choose.

## Noms CLI

Replicant is internally built on top of [Noms](https://github.com/attic-labs/noms). This is an implementation detail that we don't intend to expose to users. But while Replicant is young, it can ocassionally be useful to dive down into the guts and see what's going on.

*** Warning! The Noms CLI is extremely user-unfriendly. This is not intended to be part of the long-term DX of Replicant, it's just a temporary stop-gap. ***

Use the Noms CLI the same as `repl` - just copy it out of the release and run it.

See the [Noms CLI Tour](https://github.com/attic-labs/noms/blob/master/doc/cli-tour.md) for an introduction to using the CLI.

Here are some starting commands that will be useful to Replicant users:

```
# Prints out the current local state of the database.
# You can also ask for "remote" which is analagous to Git's origin/master -- it is the last-known state of the remote.
noms show /path/to/mydb::local

# Page through the (local) history of the database.
# Useful flags to this:
# --show-value (show the entire value of each commit, not just the diff)
# --max-lines
# --graph
# ... etc ... see --help for more.
noms log /path/to/mydb::local

# Note: there are some bugs in `noms log` where some flag combinations can cause it to crash when paging through
# logs. This doesn't indicate bad data or a deeper problem, it's just a bug in `log` :(.

# Prints out the entire current key/value store
noms show /path/to/mydb::local.value.data@target

# Prints out just the current value of "foo".
# The allowed syntax for show is fairly rich. For details, see:
# https://github.com/attic-labs/noms/blob/master/doc/spelling.md
noms show '/path/to/mydb::local.value.data@target["foo"]'

# Prints out the value of "foo" at a particular commit.
noms show '/path/to/mydb::#ks3ug9d7bavt69g6hjlssgfp6mc4scl9.value.data@target["foo"]'

# Prints the diff between two local commits (or arbitrary paths).
noms diff /path/to/mydb::#ks3ug9d7bavt69g6hjlssgfp6mc4scl9 /path/to/mydb::local

# Prints the diff between commits on different databases
noms diff https://replicate.to/serve/mydb::local /path/to/mydb::local
```
