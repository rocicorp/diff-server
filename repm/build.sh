# repm
echo "Building repm..."

set -x
ORIG=`pwd`
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT=$DIR/../
cd $ROOT
rm -rf vendor
go mod vendor > /dev/null 2>&1
cd $GOPATH/src

# Need to turn this off to build repm because Gomobile doesn't support modules,
# and as of go 1.13 the default is on if the source code contains a go.mod file,
# regardless of location.
export GO111MODULE=off

mkdir -p github.com/aboodman
ln -s $ROOT github.com/aboodman/replicant > /dev/null 2>&1 
cd github.com/aboodman/replicant/repm
rm -rf build
mkdir build
cd build
gomobile bind -ldflags="-s -w" --target=ios ../
gomobile bind -ldflags="-s -w" --target=android ../
tar -czvf Repm.framework.tar.gz Repm.framework

export GO111MODULE=
cd $ORIG

