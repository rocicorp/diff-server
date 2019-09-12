# The `rep` CLI

The Replicant SDK includes a command-line program called `rep` which you can use to inspect and manipulate
Replicant databases from the terminal.

To install it, just copy the binary for your platform to someplace in your path and run it:

```
cp <sdk>/replicant-darwin-amd64 ~/bin/rep
chmod u+x ~/bin/rep
```

## Interacting with Replicant databases

Examples:

```
# List all the items in a local database
rep --db=/path/to/my/db scan --start-at='message/'

# Execute a transaction on a remote database
rep --db=https://replicate.to/serve/my-remote-db exec sellWidgets 42
```

See `rep --help` for complete documentation.
