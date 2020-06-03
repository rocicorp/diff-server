#!/bin/sh

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT=$DIR/../
set -x

DIFFS_VERSION=`git describe --tags`

BUILDDIR=build
rm -rf build
mkdir build

cd $ROOT

# diffs
echo "Building diffs..."

GOOS=darwin GOARCH=amd64 go build -ldflags "-X roci.dev/replicant/util/version.v=$DIFFS_VERSION" -o build/diffs-osx ./cmd/diffs
GOOS=linux GOARCH=amd64 go build -ldflags "-X roci.dev/replicant/util/version.v=$DIFFS_VERSION" -o build/diffs-linux ./cmd/diffs

# noms tool
echo "Building noms..."
NOMS_VERSION=`go mod graph | grep '^github.com/attic-labs/noms@' | cut -d' ' -f1 | head -n1`
go get $NOMS_VERSION
GOOS=darwin GOARCH=amd64 go build -o build/noms-osx github.com/attic-labs/noms/cmd/noms
GOOS=linux GOARCH=amd64 go build -o build/noms-linux github.com/attic-labs/noms/cmd/noms
