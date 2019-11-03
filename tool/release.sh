#!/bin/sh

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT=$DIR/../
set -x

rm -rf build
mkdir build
cd build

# repl
echo "Building repl..."

REPL_VERSION=`git describe --tags`
cd $ROOT
GOOS=darwin GOARCH=amd64 go build -ldflags "-X roci.dev/replicant/util/version.v=$REPL_VERSION" -o build/repl-darwin-amd64 ./cmd/repl
GOOS=linux GOARCH=amd64 go build -ldflags "-X roci.dev/replicant/util/version.v=$REPL_VERSION" -o build/repl-linux-amd64 ./cmd/repl

# noms tool
echo "Building noms..."
NOMS_VERSION=`go mod graph | grep '^github.com/attic-labs/noms@' | cut -d' ' -f1 | head -n1`
go get $NOMS_VERSION
GOOS=darwin GOARCH=amd64 go build -o build/noms-darwin-amd64 github.com/attic-labs/noms/cmd/noms
GOOS=linux GOARCH=amd64 go build -o build/noms-linux-amd64 github.com/attic-labs/noms/cmd/noms
