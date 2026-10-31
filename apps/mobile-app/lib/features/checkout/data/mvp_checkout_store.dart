import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_repository.dart';

String _key(String owner, String scope) =>
    'oma_checkout_v1_${base64Url.encode(utf8.encode(jsonEncode([scope, owner])))}';

class MemoryMvpCheckoutStore implements MvpCheckoutStore {
  final _attempts = <String, MvpCheckoutAttempt>{};
  @override
  Future<MvpCheckoutAttempt?> read(String owner, String scope) async =>
      _attempts[_key(owner, scope)];
  @override
  Future<void> write(MvpCheckoutAttempt attempt) async {
    _attempts[_key(attempt.owner, attempt.scope)] = attempt;
  }
}

class SecureMvpCheckoutStore implements MvpCheckoutStore {
  const SecureMvpCheckoutStore({this._storage = const FlutterSecureStorage()});
  final FlutterSecureStorage _storage;
  @override
  Future<MvpCheckoutAttempt?> read(String owner, String scope) async {
    final value = await _storage.read(key: _key(owner, scope));
    if (value == null) return null;
    final attempt = MvpCheckoutAttempt.fromJson(
      Map<String, dynamic>.from(jsonDecode(value) as Map),
    );
    if (attempt.owner != owner || attempt.scope != scope) {
      throw const FormatException('Checkout storage ownership mismatch');
    }
    return attempt;
  }

  @override
  Future<void> write(MvpCheckoutAttempt attempt) => _storage.write(
    key: _key(attempt.owner, attempt.scope),
    value: jsonEncode(attempt.toJson()),
  );
}
