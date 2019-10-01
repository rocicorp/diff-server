import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';

const CHANNEL_NAME = 'replicant.dev';

typedef void ChangeHandler();
typedef void SyncHandler(bool syncing);

class Replicant {
  ChangeHandler onChange;
  SyncHandler onSync;

  String _name;
  String _remote;
  MethodChannel _platform;
  Future<String> _root;
  Timer _timer;

  /// Create or open a local Replicant database with named `name` synchronizing with `remote`.
  /// If `name` is omitted, it defaults to `remote`.
  Replicant(this._remote, {name = ""}) {
    if (this._remote == "") {
      throw new Exception("remote must be non-empty");
    }
    if (name == "") {
      name = this._remote;
    }
    this._name = name;

    _platform = MethodChannel(CHANNEL_NAME);
    _invoke('open');
    _root = _getRoot();
    this.sync();
  }

  /// Adds new transactions to the db.
  Future<void> putBundle(String bundle) async {
    // We check for changes here, even though putBundle doesn't change data, because
    // it can change the bundle which the client app uses to read the data, thus it
    // can affect display.
    return _result(await _checkChange(await _invoke('putBundle', {'code': bundle})));
  }

  /// Executes the named function with provided arguments from the current
  /// bundle as an atomic transaction.
  Future<dynamic> exec(String function, [List<dynamic> args = const []]) async {
    return _result(await _checkChange(await _invoke('exec', {'name': function, 'args': args})));
  }

  /// Puts a single value into the database in its own transaction.
  Future<void> put(String id, dynamic value) async {
    return _result(await _checkChange(await _invoke('put', {'id': id, 'value': value})));
  }

  /// Get a single value from the database.
  Future<dynamic> get(String id) async {
    return _result(await _invoke('get', {'id': id}));
  }

  /// Gets many values from the database.
  Future<List<ScanItem>> scan({prefix: String, startAtID: String, limit = 50}) async {
    List<Map<String, dynamic>> r = await _invoke('scan', {prefix: prefix, startAtID: startAtID, limit: limit});
    return r.map((e) => ScanItem.fromJson(e));
  }

  /// Synchronizes the database with the server. New local transactions that have been executed since the last
  /// sync are sent to the server, and new remote transactions are received and replayed.
  Future<void> sync() async {
    this._fireOnSync(true);
    try {
      if (_timer == null) {
        // Another call stack is already inside _sync();
        return;
      }

      _timer.cancel();
      _timer = null;
      await _checkChange(await _invoke("sync", {'remote': this._remote}));
    } catch (e) {
      print('ERROR DURING SYNC');
      print(e);
      // We are seeing some consistency errors during sync -- we push commits,
      // then turn around and fetch them and expect to see them, but don't.
      // that is bad, but for now, just retry.
      _timer = new Timer(new Duration(seconds: 1), sync);
    } finally {
      _timer = new Timer(new Duration(seconds: 5), sync);
      this._fireOnSync(false);
    }
  }

  Future<void> dropDatabase() async {
    return _result(await _checkChange(await _invoke('dropDatabase')));
  }

  Future<String> _getRoot() async {
    var res = await _invoke('getRoot');
    return res['root'];
  }

  dynamic _result(Map<String, dynamic> m) {
    return m == null ? null : m['result'];
  }

  Future<Map<String, dynamic>> _checkChange(Map<String, dynamic> result) async {
    var currentRoot = await _root;  // instantaneous except maybe first time
    if (result != null && result['root'] != null && result['root'] != currentRoot) {
      _root = Future.value(result['root']);
      _fireOnChange();
    }
    return result;
  }

  Future<dynamic> _invoke(String name, [Map<String, dynamic> args = const {}]) async {
    try {
      final r = await _platform.invokeMethod(name, [_name, jsonEncode(args)]);
      return r == '' ? null : jsonDecode(r);
    } catch (e) {
      throw new Exception('Error invoking "' + name + '": ' + e.toString());
    }
  }

  void _fireOnSync(bool syncing) {
    if (onSync != null) {
      scheduleMicrotask(() => onSync(syncing));
    }
  }

  void _fireOnChange() {
    if (onChange != null) {
      scheduleMicrotask(onChange);
    }
  }
}

class ScanItem {
  ScanItem.fromJson(Map<String, dynamic> data)
      : id = data['id'],
        value = data['value'] {
  }
  String id;
  var value;
}
