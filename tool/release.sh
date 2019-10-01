#!/bin/sh

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT=$DIR/../
set -x

rm -rf build
mkdir build
cd build

# repm
../repm/build.sh
cp ../repm/build/Repm.framework.tar.gz .
cp ../repm/build/repm.aar .

# Flutter
../bind/flutter/build.sh
cp ../bind/flutter/build/replicant-flutter-sdk.tar.gz .

# React Native
../bind/react-native/build.sh
cp ../bind/react-native/build/replicant-react-native.tar.gz .

# repl
echo "Building repl..."

cd $ROOT
GOOS=darwin GOARCH=amd64 go build -o build/repl-darwin-amd64 ./cmd/repl
GOOS=linux GOARCH=amd64 go build -o build/repl-linux-amd64 ./cmd/repl

# noms tool
echo "Building noms..."
NOMS_VERSION=`go mod graph | grep '^github.com/attic-labs/noms@' | cut -d' ' -f1 | head -n1`
go get $NOMS_VERSION
GOOS=darwin GOARCH=amd64 go build -o build/noms-darwin-amd64 github.com/attic-labs/noms/cmd/noms
GOOS=linux GOARCH=amd64 go build -o build/noms-linux-amd64 github.com/attic-labs/noms/cmd/noms
