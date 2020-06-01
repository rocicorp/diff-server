#!/bin/sh

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT=$DIR/../
set -x

rm -rf build
mkdir build
mkdir build/osx
mkdir build/linux

cd $ROOT

# diffs
echo "Building diffs..."

DIFFS_VERSION=`git describe --tags`
GOOS=darwin GOARCH=amd64 go build -ldflags "-X roci.dev/replicant/util/version.v=$DIFFS_VERSION" -o build/osx/diffs ./cmd/diffs
GOOS=linux GOARCH=amd64 go build -ldflags "-X roci.dev/replicant/util/version.v=$DIFFS_VERSION" -o build/linux/diffs ./cmd/diffs

# noms tool
echo "Building noms..."
NOMS_VERSION=`go mod graph | grep '^github.com/attic-labs/noms@' | cut -d' ' -f1 | head -n1`
go get $NOMS_VERSION
GOOS=darwin GOARCH=amd64 go build -o build/osx/noms github.com/attic-labs/noms/cmd/noms
GOOS=linux GOARCH=amd64 go build -o build/linux/noms github.com/attic-labs/noms/cmd/noms

mv build build-${DIFFS_VERSION}
