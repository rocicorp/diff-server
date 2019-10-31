# Local-First React Native in Less than 5 Minutes

#### 1. Get the package

Download the latest [replicant-react-native.tar.gz](https://github.com/aboodman/replicant/releases).

#### 2. Install the package

In your project's directory:

```
yarn add /path/to/replicant-react-native.tar.gz
react-native link
cd ios
pod install
```

#### 3. Create a transaction bundle

You interact with Replicant by executing _transactions_, which are written in JavaScript.

Create a new `replicant.bundle` file inside your app to hold these transactions, containing this code:

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

#### 4. Mark `*.bundle` files as assets inside `metro.config.js`:

```
...
resolver: {
  assetExts: ['png', 'jpeg', 'svg', 'bundle'],
},
...
```

#### 5. Import Replicant

```js
import Replicant from 'replicant-react-native';
```

#### 6. Instantiate Replicant

```js
var rep = Replicant('https://replicate.to/serve/any-name-here');
```

For now, you can use any name you want after `serve` in the URL.

#### 7. Register your bundle with Replicant

```js
var rep = new Replicant('https://replicate.to/serve/aa-react-native');

// Eep, so grotty. Working on it.
const resource = require('./replicant.bundle');
const resolved = Image.resolveAssetSource(resource).uri.replace('/assets', '');

await rep.putBundle(await (await fetch(resolved)).text());
```

#### 8. Execute transactions

```js
rep.onChange = () => {
  var count = await rep.exec('getCount');
  print('The answer is ${count}');
};

await rep.exec('increment', [1]);
await rep.exec('increment', [41]);
```

### Whew! All done. Time for a cup of coffee ☕️.

In fact, while you're away, why not install the app on two devices and let them sync with each other?

Disconnect them. Take a subway ride. Whatever. It's all good. The devices will sync up automatically when there is connectivity.

Conflicts are handled naturally by ordering atomic transactions consistently on all devices.
