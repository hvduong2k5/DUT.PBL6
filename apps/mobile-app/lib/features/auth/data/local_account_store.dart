import 'dart:async';
import 'dart:collection';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class LocalAccountLock {
  bool _busy = false;
  final _waiting = Queue<void Function()>();
  Future<T> run<T>(Future<T> Function() action) {
    final result = Completer<T>();
    void finish() {
      _busy = false;
      if (_waiting.isNotEmpty) _waiting.removeFirst()();
    }

    void execute() {
      _busy = true;
      Future<T>.sync(action).then(
        (value) {
          result.complete(value);
          finish();
        },
        onError: (Object error, StackTrace stack) {
          result.completeError(error, stack);
          finish();
        },
      );
    }

    // Keep each caller's zone, rather than chaining onto a foreign-zone Future.
    final task = Zone.current.bindCallback(execute);
    if (_busy) {
      _waiting.add(task);
    } else {
      task();
    }
    return result.future;
  }
}

abstract interface class LocalAccountStore {
  Future<String?> read();
  Future<void> write(String value);
  Future<T> exclusive<T>(Future<T> Function() action);
}

class SecureLocalAccountStore implements LocalAccountStore {
  const SecureLocalAccountStore({
    this.storage = const FlutterSecureStorage(
      aOptions: AndroidOptions(
        storageNamespace: 'oma_local_identity_v1',
        resetOnError: false,
      ),
    ),
  });
  static const key = 'oma_local_accounts_v1';
  static final _lock = LocalAccountLock();
  final FlutterSecureStorage storage;
  @override
  Future<String?> read() => storage.read(key: key);
  @override
  Future<void> write(String value) => storage.write(key: key, value: value);
  @override
  Future<T> exclusive<T>(Future<T> Function() action) => _lock.run(action);
}

class MemoryLocalAccountStore implements LocalAccountStore {
  String? value;
  final _lock = LocalAccountLock();
  @override
  Future<String?> read() async => value;
  @override
  Future<void> write(String value) async {
    this.value = value;
  }

  @override
  Future<T> exclusive<T>(Future<T> Function() action) => _lock.run(action);
}
