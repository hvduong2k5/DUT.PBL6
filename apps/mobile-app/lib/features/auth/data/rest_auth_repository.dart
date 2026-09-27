import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/core/storage/token_store.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';

class RestAuthRepository implements AuthRepository {
  RestAuthRepository(this._client, this._tokens);
  final ApiClient _client;
  final TokenStore _tokens;

  @override
  Future<AuthSession> login({
    required String phoneNumber,
    required String password,
  }) => _authenticate('/api/v1/auth/login', {
    'phone_number': phoneNumber,
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
    await _tokens.clear();
  }
}
