# How do I delete an entire database and its history (e.g., during development)

This is tricky because of the fact that Replicant always seeks to converge disconnected nodes.

If you delete the server database, then the first client that connects will repopulate it during sync.
If you delete a client database, the same will happen in reverse.

To delete a replicant database, you have to delete it from all the nodes in the group:

1. Uninstall the app (or clear the app's local storage) from all devices in the Replicant Group.
2. Run `repl --db=https://replicate.to/serve/<yourdb> drop` using the [Replicant CLI](cli.md).
