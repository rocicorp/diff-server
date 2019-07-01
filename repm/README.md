This is a simple mobile API to Replicant using [Gomobile](https://godoc.org/golang.org/x/mobile/cmd/gomobile).

Ideally we would use straight C for the API, [but](../repc/MOBILE_COMPLE_ARGH.md).

Build (from Replicant source dir in non-GOPATH):

```
# Establishes the vendor directory
go mod vendor
REPLICANT_SRC_DIR=`pwd`

cd $GOPATH/src
mkdir -p github.com/aboodman
ln -s $REPLICANT_SRC_DIR github.com/aboodman/replicant
cd github.com/aboodman/replicant/repm
gomobile bind . 
```
