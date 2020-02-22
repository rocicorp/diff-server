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
```

## Deploy

```
now deploy
now deploy --prod
```

... or just check in a new commit, it will autodeploy.
