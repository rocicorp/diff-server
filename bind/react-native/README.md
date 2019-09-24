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
react-native link
cd ios
pod install
```

#### 4. Create a transaction bundle

You interact with Replicant by executing JavaScript _transactions_.

Create a new `assets/replicant.bundle` file inside your app to hold these transactions, then add this code to it:

```js
function codeVersion() {
    return 1.1;
}

function increment(delta) {
    var val = getCount();
    db.put('count', val + delta);
}

function getCount() {
    return db.get('count') || 0;
}
```

#### 5. Mark `*.bundle` files as assets inside `metro.config.js`:

```
...
+  resolver: {
+    assetExts: ['bundle'],
+  },
...
```

#### 6. Instantiate Replicant

```js
var rep = Replicant('https://replicate.to/serve/any-name-here');
```

For now, you can use any name you want after `serve` in the URL.

#### 7. Register your transaction bundle

```js
const resource = require('./replicant.bundle');
const resolved = Image.resolveAssetSource(resource);
await this._replicant.putBundle(await (await fetch(resolved.uri)).text());
```

#### 8. Execute transactions

```js
await rep.exec('increment', [1]);
await rep.exec('increment', [41]);
var count = await rep.exec('getCount');
print('The answer is ${count}');
```
