#!/bin/sh
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
cp -R ../bind/flutter replicant-flutter-sdk
rm -rf replicant-flutter-sdk/ios/Repm.framework
cp -R Repm.framework replicant-flutter-sdk/ios/
cp repm.aar replicant-flutter-sdk/android/
zip -r replicant-flutter-sdk.zip replicant-flutter-sdk
