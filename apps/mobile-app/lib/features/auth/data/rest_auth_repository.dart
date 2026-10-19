import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/storage/token_store.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/domain/recovery_verification.dart';

class RestAuthRepository implements AuthRepository {
  RestAuthRepository(this._client, this._tokens);
  final ApiClient _client;
  final TokenStore _tokens;

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
    await _client.postJson('/api/v1/auth/logout');
    await _client.clearSessionCookies();
    await _tokens.clear();
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
    await _client.postJson(
      '/api/v1/auth/recovery-requests/reset',
      data: {'resetProof': proof, 'newPassword': password},
      headers: {'Idempotency-Key': idempotencyKey},
    );
    await _client.clearSessionCookies();
    await _tokens.clear();
  }
}
