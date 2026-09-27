import 'package:mobile_app/features/auth/domain/auth_models.dart';

abstract interface class AuthRepository {
  Future<AuthSession> login({
    required String phoneNumber,
    required String password,
  });
  Future<AuthSession> register({
    required String phoneNumber,
    required String password,
    required String fullName,
    String? email,
  });
  Future<void> logout();
}
