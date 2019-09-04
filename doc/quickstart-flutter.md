# Conflict-Free Offline Sync in 5 Minutes

### 1. Download and unzip the latest release

```
# Download the latest replicant-flutter-sdk.zip from https://github.com/aboodman/replicant/releases.
cd ~/Downloads
unzip replicant-flutter-sdk.zip
```

### 2. Add the `replicant` dependency to your pubspec.yaml

```
...

  cupertino_icons: ^0.1.2

+   replicant:
+     path:
+       /tmp/replicant-flutter-sdk/

...
```

### Add your transaction bundle

```
echo "function codeVersion() {
    return 1.1;
}

function incr(delta) {
    var val = getCount();
    db.put('count', val + delta);
}

function getCount() {
    return db.get('count') || 0;
}
" >> lib/bundle.js
```

### Instanciate Replicant in your app

```
var r = Replicant('https://repilicate.to/serve/any-name-here)
```

You can use any name you want currenty for the remote database name. The hosted service has no authentication yet.

### Execute transactions

```
await r.exec('incr', [42]);
var answer = await r.exec('getCount');
```

Congratulations! You are now done! Time for a cup of coffee.

In fact, while you're out, why not install the app on two devices and let them sync with each other?

Disconnect them. Take a subway ride. Whatever. It's all good. Replicant will sync whenever there is connectivity.

# Next Steps

- See [`hello`](../samples/flutter/hello), which contains the above steps in a runnable app
- See [`replido`](../samples/flutter/replido) a fully functioning TODO app built on Flutter and Replicant
- See the full Dart API for Replicant
- Understand the JavaScript API available inside Replicant transactions
- Inspect your Replicant databases using the `rep` tool

# More Questions?

See the [design doc](../README.md).
