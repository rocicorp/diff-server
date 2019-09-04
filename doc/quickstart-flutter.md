# Conflict-Free Offline Sync in Less than 5 Minutes

### 1. Download and unzip the latest release

Get the latest replicant-flutter-sdk.zip from https://github.com/aboodman/replicant/releases then:

```
unzip replicant-flutter-sdk.zip
```

### 2. Add the `replicant` dependency to your `pubspec.yaml`

```
...

  cupertino_icons: ^0.1.2

+   replicant:
+     path:
+       /tmp/replicant-flutter-sdk/

...
```

### 3. Add your transaction bundle

Create a new `lib/bundle.js` file inside your app, and add this code to it:

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

### 4. Mark `lib/bundle.js` as an asset inside `pubspec.yaml`:

```
...

flutter:
  uses-material-design: true
  assets:
+    - lib/bundle.js

...
```

### 5. Instantiate Replicant in your app

```
var rep = Replicant('https://repilicate.to/serve/any-name-here')
```

For now, you can use any name you want after `serve` in the URL.

### 6. Execute transactions

```
await rep.exec('incr', [1]);
await rep.exec('incr', [41]);
var count = await rep.exec('getCount');
print('The answer is ${count}');
```

Congratulations — you are done 🎉. Time for a cup of coffee.

In fact, while you're away, why not install the app on two devices and let them sync with each other?

Disconnect them. Take a subway ride. Whatever. It's all good. Replicant will sync whenever there is connectivity.

# Want Something even Easier?

Download the above steps as a running sample. See [flutter/hello](../samples/flutter/hello).

# Next Steps

- See [`replido`](../samples/flutter/replido) a fully functioning TODO app built on Flutter and Replicant
- Review the Dart API for Replicant
- Review the JavaScript API for Replicant transactions
- Inspect your Replicant databases using the `rep` tool

# More Questions?

See the [design doc](../README.md).
