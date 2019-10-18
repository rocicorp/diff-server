import 'dart:core';
import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';

import 'database_info.dart';

const CHANNEL_NAME = 'replicant.dev';

typedef void ChangeHandler();
typedef void SyncHandler(bool syncing);

/// Replicant is a connection to a local Replicant database. There can be multiple
/// connections to the same database.
/// 
/// Operations are generally async because they go to local storage. However on modern
/// mobile devices this will typically be ~instant, and in most cases no progress UI
/// should be necessary.
/// 
/// Replicant operations are serialized per-connection, with the sole exception of
/// sync(), which runs concurrently with other operations (and might take awhile, since
/// it attempts to go to the network).
class Replicant {
  static MethodChannel _platform = MethodChannel(CHANNEL_NAME);

  ChangeHandler onChange;
  SyncHandler onSync;

  // If true, Replicant only syncs the head of the remote repository, which is
  // must faster. Currently this disables bidirectional sync though :(.
  bool hackyShallowSync;

  String _name;
  String _remote;
  Future<String> _root;
  Timer _timer;
  bool _closed = false;

  /// Lists information about available local databases.
  static Future<List<DatabaseInfo>> list() async {
    var res = await _invoke('', 'list');
    return List.from(res['databases'].map((d) => DatabaseInfo.fromJSON(d)));
  }

  /// Completely delete a local database. Remote replicas in the group aren't affected.
  static Future<void> drop(String name) async {
    await _invoke(name, 'drop');
  }

  /// Create or open a local Replicant database with named `name` synchronizing with `remote`.
  /// If `name` is omitted, it defaults to `remote`.
  Replicant(this._remote, {String name = ""}) {
    if (this._remote == "") {
      throw new Exception("remote must be non-empty");
    }
    if (name == "") {
      name = this._remote;
    }
    this._name = name;
    _invoke(this._name, 'open');
    _root = _getRoot();
    this._scheduleSync(0);
  }

  String get name => _name;
  String get remote => _remote;

  /// Adds new transactions to the db.
  Future<void> putBundle(String bundle) async {
    // We check for changes here, even though putBundle doesn't change data, because
    // it can change the bundle which the client app uses to read the data, thus it
    // can affect display.
    return _result(await _checkChange(await _invoke(this._name, 'putBundle', {'code': bundle})));
  }

  /// Executes the named function with provided arguments from the current
  /// bundle as an atomic transaction.
  Future<dynamic> exec(String function, [List<dynamic> args = const []]) async {
    return _result(await _checkChange(await _invoke(this._name, 'exec', {'name': function, 'args': args})));
  }

  /// Puts a single value into the database in its own transaction.
  Future<void> put(String id, dynamic value) async {
    return _result(await _checkChange(await _invoke(this._name, 'put', {'id': id, 'value': value})));
  }

  /// Get a single value from the database.
  Future<dynamic> get(String id) async {
    return _result(await _invoke(this._name, 'get', {'id': id}));
  }

  /// Gets many values from the database.
  Future<List<ScanItem>> scan({prefix: String, startAtID: String, limit = 50}) async {
    List<Map<String, dynamic>> r = await _invoke(this._name, 'scan', {prefix: prefix, startAtID: startAtID, limit: limit});
    return r.map((e) => ScanItem.fromJson(e));
  }

  /// Synchronizes the database with the server. New local transactions that have been executed since the last
  /// sync are sent to the server, and new remote transactions are received and replayed.
  Future<void> sync() async {
    if (_closed) {
      return;
    }

    this._fireOnSync(true);
    try {
      if (_timer == null) {
        // Another call stack is already inside _sync();
        return;
      }

      _timer.cancel();
      _timer = null;
      await _checkChange(await _invoke(this._name, "sync", {'remote': this._remote, 'shallow': this.hackyShallowSync}));
      _scheduleSync(5);
    } catch (e) {
      // We are seeing some consistency errors during sync -- we push commits,
      // then turn around and fetch them and expect to see them, but don't.
      // that is bad, but for now, just retry.
      print('ERROR DURING SYNC');
      print(e);
      _scheduleSync(1);
    } finally {
      this._fireOnSync(false);
    }
  }

  void _scheduleSync(seconds) {
      _timer = new Timer(new Duration(seconds: seconds), sync);
  }

  Future<void> close() async {
    _closed = true;
    await _invoke(this.name, 'close');
  }

  Future<String> _getRoot() async {
    var res = await _invoke(this._name, 'getRoot');
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

  static Future<dynamic> _invoke(String dbName, String rpc, [Map<String, dynamic> args = const {}]) async {
    try {
      final r = await _platform.invokeMethod(rpc, [dbName, jsonEncode(args)]);
      return r == '' ? null : jsonDecode(r);
    } catch (e) {
      throw new Exception('Error invoking "' + rpc + '": ' + e.toString());
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
