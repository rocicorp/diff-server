#!/bin/sh

# Need to turn this off to build repm because Gomobile doesn't support modules,
# and as of go 1.13 the default is on if the source code contains a go.mod file,
# regardless of location.
export GO111MODULE=off

# repm
ORIG=`pwd`
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT=$DIR/../
cd $ROOT
set -x
go mod vendor > /dev/null 2>&1
cd $GOPATH/src
mkdir -p github.com/aboodman
ln -s $ROOT github.com/aboodman/replicant > /dev/null 2>&1 
cd github.com/aboodman/replicant
rm -rf build
mkdir build
cd build
gomobile bind -ldflags="-s -w" --target=ios ../repm/
gomobile bind -ldflags="-s -w" --target=android ../repm/
zip -r Repm.framework.zip Repm.framework

# flutter bindings
cp -R ../bind/flutter replicant-flutter-sdk
rm -rf replicant-flutter-sdk/ios/Repm.framework
cp -R Repm.framework replicant-flutter-sdk/ios/
cp repm.aar replicant-flutter-sdk/android/
zip -r replicant-flutter-sdk.zip replicant-flutter-sdk

# rep tool

# turn modules back on to build cli :(
export GO111MODULE=on

cd $ROOT
GOOS=darwin GOARCH=amd64 go build -o build/rep-darwin-amd64 ./cmd/rep
GOOS=linux GOARCH=amd64 go build -o build/rep-linux-amd64 ./cmd/rep
