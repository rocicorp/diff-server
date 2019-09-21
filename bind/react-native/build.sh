#!/bin/sh
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
set -x
rm -rf build
mkdir build
mkdir build/replicant-react-native
ls | grep -v build | xargs -I{} cp -R {} build/replicant-react-native/
mkdir build/replicant-react-native/ios/Frameworks
cp -R $DIR/../../build/Repm.framework $DIR/build/replicant-react-native/ios/Frameworks/
cd build
tar -czvf replicant-react-native.tar.gz replicant-react-native
cd -

