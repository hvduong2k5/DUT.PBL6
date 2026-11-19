import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:mobile_app/features/orders/domain/mvp_order_models.dart';
import 'package:mobile_app/features/orders/domain/mvp_orders_repository.dart';

String _key(String owner, String scope, String id) =>
    'oma_cancel_v1_${base64Url.encode(utf8.encode(jsonEncode([scope, owner, id])))}';

class MemoryMvpCancellationStore implements MvpCancellationStore {
  final _records = <String, MvpCancellationCommand>{};
  @override
  Future<MvpCancellationCommand?> read(
    String owner,
    String scope,
    String orderId,
  ) async => _records[_key(owner, scope, orderId)];
  @override
  Future<void> write(MvpCancellationCommand command) async {
    _records[_key(command.owner, command.scope, command.orderId)] = command;
  }
}

class SecureMvpCancellationStore implements MvpCancellationStore {
  const SecureMvpCancellationStore({
    this.storage = const FlutterSecureStorage(),
  });
  final FlutterSecureStorage storage;
  @override
  Future<MvpCancellationCommand?> read(
    String owner,
    String scope,
    String orderId,
  ) async {
    final value = await storage.read(key: _key(owner, scope, orderId));
    if (value == null) return null;
    final command = MvpCancellationCommand.fromJson(
      Map<String, dynamic>.from(jsonDecode(value) as Map),
    );
    if (command.owner != owner ||
        command.scope != scope ||
        command.orderId != orderId) {
      throw const FormatException('Cancellation ownership mismatch');
    }
    return command;
  }

  @override
  Future<void> write(MvpCancellationCommand command) => storage.write(
    key: _key(command.owner, command.scope, command.orderId),
    value: jsonEncode(command.toJson()),
  );
}
