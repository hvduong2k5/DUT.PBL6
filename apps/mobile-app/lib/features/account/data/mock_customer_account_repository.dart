import 'dart:async';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/account/domain/customer_account_repository.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';

class MockCustomerAccountRepository implements CustomerAccountRepository {
  const MockCustomerAccountRepository({
    required this.currentUser,
    required this.onUpdate,
  });
  final AuthUser? Function() currentUser;
  final FutureOr<void> Function(AuthUser) onUpdate;
  AuthUser _user() {
    final user = currentUser();
    if (user == null) {
      throw const ApiException(
        Failure(
          kind: FailureKind.unauthenticated,
          message: 'Vui lòng đăng nhập để xem thông tin tài khoản.',
        ),
      );
    }
    return user;
  }

  @override
  Future<CustomerProfile> profile() async {
    await Future<void>.delayed(const Duration(milliseconds: 180));
    return CustomerProfile(user: _user(), tier: 'BRONZE', points: 450);
  }

  @override
  Future<CustomerProfile> updateProfile(String fullName, String email) async {
    final current = _user();
    await Future<void>.delayed(const Duration(milliseconds: 200));
    final user = AuthUser(
      id: current.id,
      phoneNumber: current.phoneNumber,
      fullName: fullName,
      email: email,
    );
    await onUpdate(user);
    return CustomerProfile(user: user, tier: 'BRONZE', points: 450);
  }

  @override
  Future<LoyaltyOverview> loyalty() async {
    await Future<void>.delayed(const Duration(milliseconds: 180));
    return LoyaltyOverview(
      customerId: _user().id,
      points: 450,
      tier: 'BRONZE',
      vndRate: 1000,
    );
  }
}
