# replicant-react-native

## Installation

#### 1. Get the package

Download the latest [replicant-react-native.tar.gz](https://github.com/aboodman/replicant/releases).

#### 2. Install the package

In your project directory:

```
yarn add /path/to/replicant-react-native.tar.gz
```

#### 3. Add the native dependency

```
$ react-native link
$ cd ios
$ pod install
```

#### 4. Build / run the project as normal

```
$ react-native run-ios
```

## Usage
```javascript
import Replicant from 'replicant-react-native';

var rep = new Replicant('https://replicate.to/serve/any-name-you-want');
await rep.putBundle(myBundle);
await rep.exec('hello', ['foo', 'bar', 42]);
```
