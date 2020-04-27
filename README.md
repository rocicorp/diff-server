![Go](https://github.com/rocicorp/diff-server/workflows/Go/badge.svg)

# Replicache Diff Server

This repository implements the Replicache Diff Server. See [Replicache](https://github.com/rocicorp/replicache) for more information.

## Build

```
cd ~/work
git clone https://github.com/rocicorp/diff-server
cd diff-server
go build ./cmd/diffs
go test ./...
```

## Run (Development Mode)

```
./diffs serve --db=/tmp/diffs-data

curl -d '{"accountID":"sandbox", "clientID":"c1", "baseStateID":"00000000000000000000000000000000", "checksum":"00000000"}' http://localhost:7001/pull

curl -d '{"accountID":"sandbox", "clientID":"c1", "clientViewResponse":{"clientView":{"foo":"bar"},"lastTransactionID":"2"}}' http://localhost:7001/inject
```

## Deploy

```
now deploy
now deploy --prod
```

... or just check in a new commit, it will autodeploy.
