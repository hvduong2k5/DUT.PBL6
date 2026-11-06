import 'dart:async';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

Future<void> testExecutable(FutureOr<void> Function() testMain) async {
  TestWidgetsFlutterBinding.ensureInitialized();
  // Unit/widget tests use isolated platform storage; native QA tests the real vault.
  setUp(() => FlutterSecureStorage.setMockInitialValues({}));
  await testMain();
}
