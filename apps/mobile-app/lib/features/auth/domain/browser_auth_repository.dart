import 'auth_models.dart';

abstract interface class BrowserAuthRepository {
  Set<String> get providers;
  Future<AuthSession> signIn({String? provider});
  Future<void> registerCustomer({
    required String firstName,
    required String lastName,
    required String email,
    required String password,
    required String idempotencyKey,
  });
  Future<void> changePassword({
    required String currentPassword,
    required String newPassword,
  });
}
