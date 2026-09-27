import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/auth/domain/auth_repository.dart';

class MockAuthRepository implements AuthRepository {
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
}
