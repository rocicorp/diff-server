# Conflict-Free Offline Sync in Less than 5 Minutes

#### 1. Get the SDK

Download the latest [replicant-flutter-sdk.zip](https://github.com/aboodman/replicant/releases), then unzip it.

```
unzip replicant-flutter-sdk.zip
```

#### 2. Add the `replicant` dependency to your `pubspec.yaml`

```
...

  cupertino_icons: ^0.1.2

+   replicant:
+     path:
+       /tmp/replicant-flutter-sdk/

...
```

#### 3. Create a transaction bundle

You interact with Replicant by executing JavaScript _transactions_.

Create a new `lib/bundle.js` file inside your app to hold these transactions, then add this code to it:

```
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

#### 4. Mark `lib/bundle.js` as an asset inside `pubspec.yaml`:

```
...

flutter:
  uses-material-design: true
  assets:
+    - lib/bundle.js

...
```

#### 5. Instantiate Replicant

```
var rep = Replicant('https://repilicate.to/serve/any-name-here')
```

For now, you can use any name you want after `serve` in the URL.

#### 6. Put bundle

```dart
await rep.putBundle(
  await rootBundle.loadString('lib/bundle.js', cache: false),
);
```

#### 7. Execute transactions

```
await rep.exec('incr', [1]);
await rep.exec('incr', [41]);
var count = await rep.exec('getCount');
print('The answer is ${count}');
```

Congratulations — you are done 🎉. Time for a cup of coffee.

In fact, while you're away, why not install the app on two devices and let them sync with each other?

Disconnect them. Take a subway ride. Whatever. It's all good. The devices will sync up automatically when there is connectivity.

[Conflicts are handled naturally](https://github.com/aboodman/replicant/blob/master/README.md#conflicts) by ordering atomic transactions consistently on all devices.

## Want something even easier?

Download the above steps as a running sample. See [flutter/hello](../samples/flutter/hello).

## Next steps

- See [`flutter/redo`](../samples/flutter/redo) a fully functioning TODO app built on Flutter and Replicant
- Review the [Flutter API](https://replicate.to/doc/flutter/)
- Review the [JavaScript API for Replicant transactions](transaction-api.md)
- Inspect your Replicant databases using [the `rep` tool](cli.md)

## More questions?

See the [design doc](../README.md).
