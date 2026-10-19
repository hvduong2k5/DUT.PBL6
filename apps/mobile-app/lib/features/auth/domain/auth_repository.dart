import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/recovery_verification.dart';

abstract interface class AuthRepository {
  Future<AuthUser?> restoreSession();
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
  Future<void> requestRecovery(String email, {required String idempotencyKey});
  Future<RecoveryVerification> verifyRecovery(
    String token, {
    required String idempotencyKey,
  });
  Future<void> resetPassword(
    String proof,
    String password, {
    required String idempotencyKey,
  });
}
