import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/storage/token_store.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/domain/recovery_verification.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:uuid/uuid.dart';

class RestAuthRepository implements AuthRepository {
  RestAuthRepository(this._client, this._tokens);
  final ApiClient _client;
  final TokenStore _tokens;
  String? _refreshKeyToken, _refreshKey;
  Future<AuthUser?>? _restoreInFlight;

  @override
  Future<AuthUser?> restoreSession() => _restoreInFlight ??= _restoreSession()
      .whenComplete(() => _restoreInFlight = null);

  Future<void> _waitForRestore() async {
    try {
      await _restoreInFlight;
    } catch (_) {
      /* A new explicit auth action can recover a failed restore. */
    }
  }

  Future<AuthUser?> _restoreSession() async {
    final access = await _tokens.readAccessToken();
    final refresh = await _tokens.readRefreshToken();
    if ((access == null || access.isEmpty) &&
        (refresh == null || refresh.isEmpty)) {
      return null;
    }
    try {
      return AuthUser.fromJson(await _client.getJson('/api/v1/auth/me'));
    } on ApiException catch (error) {
      if (error.failure.kind != FailureKind.unauthenticated) rethrow;
      if (refresh == null || refresh.isEmpty) {
        await _clearLocalSession();
        return null;
      }
    }
    try {
      if (_refreshKeyToken != refresh) {
        _refreshKeyToken = refresh;
        _refreshKey = const Uuid().v4();
      }
      final json = await _client.postJson(
        '/api/v1/auth/refresh',
        data: {'refresh_token': refresh},
        headers: {'Idempotency-Key': _refreshKey},
      );
      final newAccess = json['access_token'] as String,
          newRefresh = json['refresh_token'] as String;
      if (newAccess.isEmpty || newRefresh.isEmpty) {
        throw const FormatException('Empty refreshed session');
      }
      await _tokens.save(access: newAccess, refresh: newRefresh);
      return AuthUser.fromJson(await _client.getJson('/api/v1/auth/me'));
    } on ApiException catch (error) {
      if (error.failure.kind != FailureKind.unauthenticated) rethrow;
      await _clearLocalSession();
      return null;
    }
  }

  Future<void> _clearLocalSession() async {
    _refreshKey = null;
    _refreshKeyToken = null;
    await _client.clearSessionCookies();
    await _tokens.clear();
  }

  @override
  Future<AuthSession> login({
    required String phoneNumber,
    required String password,
  }) => _authenticate('/api/v1/auth/login', {
    'identifier': phoneNumber,
    'password': password,
  });

  @override
  Future<AuthSession> register({
    required String phoneNumber,
    required String password,
    required String fullName,
    String? email,
  }) => _authenticate('/api/v1/auth/register', {
    'phone_number': phoneNumber,
    'password': password,
    'full_name': fullName,
    if (email != null && email.isNotEmpty) 'email': email,
  });

  Future<AuthSession> _authenticate(
    String path,
    Map<String, dynamic> body,
  ) async {
    await _waitForRestore();
    final json = await _client.postJson(path, data: body);
    final tokenJson = Map<String, dynamic>.from(
      json['tokens'] as Map? ?? const {},
    );
    final session = AuthSession(
      user: AuthUser.fromJson(
        Map<String, dynamic>.from(json['user'] as Map? ?? const {}),
      ),
      accessToken: tokenJson['access_token'] as String? ?? '',
      refreshToken: tokenJson['refresh_token'] as String? ?? '',
    );
    await _tokens.save(
      access: session.accessToken,
      refresh: session.refreshToken,
    );
    return session;
  }

  @override
  Future<void> logout() async {
    await _waitForRestore();
    try {
      await _client.postJson('/api/v1/auth/logout');
    } finally {
      await _clearLocalSession();
    }
  }

  @override
  Future<void> requestRecovery(
    String email, {
    required String idempotencyKey,
  }) async {
    await _client.postJson(
      '/api/v1/auth/recovery-requests',
      data: {'email': email},
      headers: {'Idempotency-Key': idempotencyKey},
    );
  }

  @override
  Future<RecoveryVerification> verifyRecovery(
    String token, {
    required String idempotencyKey,
  }) async {
    final json = await _client.postJson(
      '/api/v1/auth/recovery-requests/verify',
      data: {'recoveryToken': token},
      headers: {'Idempotency-Key': idempotencyKey},
    );
    final proof = json['resetProof'] as String? ?? '';
    final expiry = DateTime.tryParse(json['expiresAt'] as String? ?? '');
    if (json['verified'] != true || proof.isEmpty || expiry == null) {
      throw const FormatException('Invalid recovery verification response');
    }
    return RecoveryVerification(resetProof: proof, expiresAt: expiry);
  }

  @override
  Future<void> resetPassword(
    String proof,
    String password, {
    required String idempotencyKey,
  }) async {
    await _waitForRestore();
    await _client.postJson(
      '/api/v1/auth/recovery-requests/reset',
      data: {'resetProof': proof, 'newPassword': password},
      headers: {'Idempotency-Key': idempotencyKey},
    );
    await _client.clearSessionCookies();
    await _tokens.clear();
  }
}
