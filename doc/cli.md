# The `rep` CLI

The Replicant SDK includes a command-line program called `rep` which you can use to inspect and manipulate
Replicant databases from the terminal.

To install it, just copy the binary for your platform to someplace in your path and run it:

```
cp <sdk>/replicant-darwin-amd64 ~/bin/rep
chmod u+x ~/bin/rep
```

## Interacting with Replicant databases

Example:

```
$ rep --db=/tmp/mydb bundle put <<HERE
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

$ rep --db=/tmp/mydb exec createUser `uuidgen` Abby orange
$ rep --db=/tmp/mydb exec createUser `uuidgen` Aaron orange
$ rep --db=/tmp/mydb exec createUser `uuidgen` Sam green

$ rep --db=/tmp/mydb exec getUsersByColor orange
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

See `rep --help` for complete documentation.

## Running a Development Server

You can run a fully-functioning Replicant server against local disk using `rep`:

```
rep --db=~/replicant-storage serve
```

You can then point your Replicant clients (the CLI, using the `--db` flag, as well as the bindings) at http://localhost:7001/foo, where `foo` is a unique database name that you choose.
