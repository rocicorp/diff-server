![Go](https://github.com/rocicorp/diff-server/workflows/Go/badge.svg)

# Replicache Diff Server

This repository implements the Replicache Diff Server. See [Replicache](https://github.com/rocicorp/replicache) for more information. See the [contributing guide](https://github.com/rocicorp/replicache/blob/master/contributing.md) there for contributing information.

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
./diffs serve --db=/tmp/diffs-data --enable-inject

curl -d '{"accountID":"sandbox", "clientID":"c1", "baseStateID":"00000000000000000000000000000000", "checksum":"00000000"}' http://localhost:7001/pull

curl -d '{"accountID":"sandbox", "clientID":"c1", "clientViewResponse":{"clientView":{"foo":"bar"},"lastTransactionID":"2"}}' http://localhost:7001/inject
```

## Deploy

```
now deploy
now deploy --prod
```

... or just check in a new commit, it will autodeploy.

## Release

1. Tag a new release eg v1.0.3:
```
git checkout master
git tag v1.0.3
git push origin
```
2. Build release binaries:
```
./tools/release.sh
```
3. Find the new tag on [https://github.com/rocicorp/diff-server/releases](https://github.com/rocicorp/diff-server/releases) and edit it.
4. Upload `diffs` and `noms` artifacts generated in previous step (found in `build/`).
5. Save the release.

Done. Customers can now run `tools/build.sh` to get the new version [as described here](https://github.com/rocicorp/replicache-sdk-js#get-binaries).
