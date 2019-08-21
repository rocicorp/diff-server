#!/bin/sh
ORIG=`pwd`
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT=$DIR/../
cd $ROOT
go mod vendor
cd $GOPATH/src
mkdir -p github.com/aboodman
ln -s $ROOT github.com/aboodman/replicant
cd github.com/aboodman/replicant/repm
rm repm.aar
rm Repm.framework.zip
rm -rf Repm.framework
gomobile bind --target=ios
gomobile bind --target=android
zip -r Repm.framework.zip Repm.framework
