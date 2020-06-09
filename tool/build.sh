ORIG=`pwd`
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT=$DIR/../
REPM_VERSION='7dc06be02dc06caf8f91b26ad06c7f9f4d9bfb1b'

echo "Building Repm Module..."

(
  cd $ROOT
  set -x

  rm -rf build
  mkdir build
  cd build


  # Build repm test_server  
  git clone https://github.com/rocicorp/replicache-client
  cd replicache-client
  git reset --hard $REPM_VERSION
  GOARCH=amd64 GOOS=darwin go build -o ../repc-amd64-osx ./cmd/test_server
  GOARCH=amd64 GOOS=linux go build -o ../repc-amd64-linux ./cmd/test_server

  cd ..
  rm -rf replicache-client
  tar cvzf replicache-sdk-js.tar.gz repc-amd64-osx repc-amd64-linux
)
