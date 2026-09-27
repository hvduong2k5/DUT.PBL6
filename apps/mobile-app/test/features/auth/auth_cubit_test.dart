import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/features/auth/data/mock_auth_repository.dart';
import 'package:mobile_app/features/auth/presentation/auth_cubit.dart';

void main() {
  test('guest session remains available when validation fails', () async {
    final cubit = AuthCubit(MockAuthRepository());
    expect(await cubit.login('', ''), isFalse);
    expect(cubit.state.status, AuthStatus.failure);
    expect(cubit.state.user, isNull);
    await cubit.close();
  });

  test('valid login establishes an authenticated session', () async {
    final cubit = AuthCubit(MockAuthRepository());
    expect(await cubit.login('0901234567', 'MatKhau@2026'), isTrue);
    expect(cubit.state.status, AuthStatus.authenticated);
    expect(cubit.state.user?.phoneNumber, '0901234567');
    await cubit.close();
  });

  test('register validates Vietnamese phone and password length', () async {
    final cubit = AuthCubit(MockAuthRepository());
    expect(
      await cubit.register(name: 'An', phone: '123', password: 'short'),
      isFalse,
    );
    expect(cubit.state.status, AuthStatus.failure);
    await cubit.close();
  });
}
