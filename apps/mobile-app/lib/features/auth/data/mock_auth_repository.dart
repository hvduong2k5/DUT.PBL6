import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';
import 'package:mobile_app/features/auth/domain/recovery_verification.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:uuid/uuid.dart';

class MockAuthRepository implements AuthRepository {
  final Map<String, DateTime> _proofs = {};
  @override
  Future<AuthSession> login({
    required String phoneNumber,
    required String password,
  }) async {
    await Future<void>.delayed(const Duration(milliseconds: 250));
    if (password == 'sai') throw StateError('Thông tin đăng nhập chưa đúng.');
    return AuthSession(
      user: AuthUser(
        id: 'CUST-MOCK',
        phoneNumber: phoneNumber,
        fullName: 'Nguyễn Văn An',
      ),
      accessToken: 'mock-access',
      refreshToken: 'mock-refresh',
    );
  }

  @override
  Future<AuthSession> register({
    required String phoneNumber,
    required String password,
    required String fullName,
    String? email,
  }) async {
    await Future<void>.delayed(const Duration(milliseconds: 250));
    return AuthSession(
      user: AuthUser(
        id: 'CUST-MOCK',
        phoneNumber: phoneNumber,
        fullName: fullName,
        email: email,
      ),
      accessToken: 'mock-access',
      refreshToken: 'mock-refresh',
    );
  }

  @override
  Future<void> logout() async {}

  @override
  Future<void> requestRecovery(
    String email, {
    required String idempotencyKey,
  }) async {
    await Future<void>.delayed(const Duration(milliseconds: 250));
    // Neutral for all emails; this adapter never sends mail.
  }

  @override
  Future<RecoveryVerification> verifyRecovery(
    String token, {
    required String idempotencyKey,
  }) async {
    await Future<void>.delayed(const Duration(milliseconds: 250));
    if (token != 'mock-token') {
      throw const ApiException(
        Failure(
          kind: FailureKind.expired,
          code: 'RECOVERY_LINK_EXPIRED',
          message: 'Liên kết không hợp lệ hoặc đã hết hạn.',
        ),
      );
    }
    final proof = const Uuid().v4();
    final expiry = DateTime.now().add(const Duration(minutes: 5));
    _proofs[proof] = expiry;
    return RecoveryVerification(resetProof: proof, expiresAt: expiry);
  }

  @override
  Future<void> resetPassword(
    String proof,
    String password, {
    required String idempotencyKey,
  }) async {
    await Future<void>.delayed(const Duration(milliseconds: 250));
    final expiry = _proofs.remove(proof);
    if (expiry == null || !expiry.isAfter(DateTime.now())) {
      throw const ApiException(
        Failure(
          kind: FailureKind.expired,
          message: 'Liên kết đã hết hạn. Vui lòng gửi yêu cầu mới.',
        ),
      );
    }
  }
}
