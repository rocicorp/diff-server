import 'dart:convert';

import 'package:flutter/services.dart';

const CHANNEL_NAME = 'replicant.dev';

class Replicant {
  MethodChannel _platform;

  Replicant() {
     _platform = MethodChannel(CHANNEL_NAME);
  }

  Future<void> putBundle(String bundle) {
    return _invoke('putBundle', {'code': bundle});
  }

  // Executes the named function with provided arguments from the current
  // bundle as an atomic transaction.
  Future<dynamic> exec(String function, [List<dynamic> args = const []]) {
    return _invoke('exec', {'name': function, 'args': args});
  }

  // Puts a single value into the database in its own transaction.
  Future<void> put(String id, dynamic value) {
    return _invoke('put', {'id': id, 'value': value});
  }

  // Get a single value from the database.
  Future<dynamic> get(String id) {
    return _invoke('get', {'id': id});
  }

  Future<void> sync(String remote) {
    return _invoke("sync", {'remote': remote});
  }

  Future<void> dropDatabase() {
    return _invoke('dropDatabase');
  }

  Future<dynamic> _invoke(String name, [Map<String, dynamic> args = const {}]) async {
    final r = await _platform.invokeMethod(name, jsonEncode(args));
    return r == '' ? null : jsonDecode(r)['result'];
  }
}
