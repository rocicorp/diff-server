# HTTP API

Replicant has a basic HTTP API that allows you to interact with a running Replicant server over HTTP.

## Details

The HTTP API is comprised of a number of _commands_. Each command is accessed via HTTP `POST` at a URL of
the form `root/<command>`. The request and response `Content-type` is `application/json`.

## Commands

For a listing of the currently available commands, see `commands` in [`serve.go`](https://github.com/aboodman/replicant/blob/master/serve/serve.go#L27).

For the detailed request and response payload on each available command, see [api.go](https://github.com/aboodman/replicant/blob/master/api/api.go#L20).

## Examples

```
# Execute a transaction on 'mydb' on replicate.to
curl --data '{"name": "addTodo", "args": [42, "Take the trash out", false]}' https://replicate.to/serve/mydb/exec

# Scan a key range on a local server
curl --data '{"prefix": "todo/", "limit": 500}' http://localhost:7001/scan
```

## Development Server

See the [CLI documentation](cli.md#running-a-development-server).
