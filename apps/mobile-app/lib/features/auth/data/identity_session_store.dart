import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class IdentitySessionRecord {
  const IdentitySessionRecord({
    required this.accessToken,
    required this.refreshToken,
    this.sessionId,
  });
  final String accessToken, refreshToken;
  final String? sessionId;
  factory IdentitySessionRecord.fromJson(Map<String, dynamic> json) {
    final access = json['accessToken'],
        refresh = json['refreshToken'],
        cookie = json['sessionId'];
    if (access is! String ||
        !RegExp(r'^[A-Za-z0-9._~+/%=-]+$').hasMatch(access) ||
        refresh is! String ||
        !RegExp(r'^[A-Za-z0-9._~+/%=-]+$').hasMatch(refresh) ||
        cookie != null &&
            (cookie is! String ||
                !RegExp(r'^[A-Za-z0-9._~+/%=-]+$').hasMatch(cookie))) {
      throw const FormatException('Invalid Identity session');
    }
    return IdentitySessionRecord(
      accessToken: access,
      refreshToken: refresh,
      sessionId: cookie as String?,
    );
  }
  Map<String, dynamic> toJson() => {
    'accessToken': accessToken,
    'refreshToken': refreshToken,
    'sessionId': sessionId,
  };
}

abstract interface class IdentitySessionStore {
  Future<IdentitySessionRecord?> read();
  Future<void> write(IdentitySessionRecord? record);
}

class SecureIdentitySessionStore implements IdentitySessionStore {
  SecureIdentitySessionStore(this.scope, {FlutterSecureStorage? storage})
    : _storage =
          storage ??
          const FlutterSecureStorage(
            aOptions: AndroidOptions(
              storageNamespace: 'oma_identity_session_v1',
              resetOnError: false,
            ),
          );
  final String scope;
  final FlutterSecureStorage _storage;
  static const _key = 'identity_session_v1';
  @override
  Future<IdentitySessionRecord?> read() async {
    final raw = await _storage.read(key: _key);
    if (raw == null) return null;
    final json = jsonDecode(raw);
    if (json is! Map || json['version'] != 1 || json['scope'] is! String) {
      throw const FormatException('Invalid Identity vault');
    }
    if (json['scope'] != scope) return null;
    if (json['session'] == null) return null;
    return IdentitySessionRecord.fromJson(
      Map<String, dynamic>.from(json['session'] as Map),
    );
  }

  @override
  Future<void> write(IdentitySessionRecord? record) => _storage.write(
    key: _key,
    value: jsonEncode({
      'version': 1,
      'scope': scope,
      'session': record?.toJson(),
    }),
  );
}

class MemoryIdentitySessionStore implements IdentitySessionStore {
  IdentitySessionRecord? record;
  @override
  Future<IdentitySessionRecord?> read() async => record;
  @override
  Future<void> write(IdentitySessionRecord? record) async {
    this.record = record;
  }
}
