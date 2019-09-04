import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:replicant/replicant.dart';

void main() {
  const MethodChannel channel = MethodChannel('replicant');

  setUp(() {
    channel.setMockMethodCallHandler((MethodCall methodCall) async {
      return '42';
    });
  });

  tearDown(() {
    channel.setMockMethodCallHandler(null);
  });

  test('getPlatformVersion', () async {
    expect(await Replicant.platformVersion, '42');
  });
}
