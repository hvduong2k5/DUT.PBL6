import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/orders/domain/guest_orders_repository.dart';
import 'package:mobile_app/features/orders/data/mock_orders_repository.dart';

class MockGuestOrdersRepository implements GuestOrdersRepository {
  MockGuestOrdersRepository(this._orders);
  final MockOrdersRepository _orders;
  final Map<String, ({String order, String contact, DateTime expires})>
  _challenges = {};
  @override
  Future<GuestOrderChallenge> challenge(
    String orderNumber,
    String contact, {
    required String idempotencyKey,
  }) async {
    await Future<void>.delayed(const Duration(milliseconds: 200));
    final id = const Uuid().v4();
    final expiry = DateTime.now().add(const Duration(minutes: 5));
    _challenges[id] = (order: orderNumber, contact: contact, expires: expiry);
    return GuestOrderChallenge(
      id: id,
      maskedDestination: 'Thông tin liên hệ đã cung cấp',
      expiresAt: expiry,
      resendAfterSeconds: 30,
      message:
          'Nếu thông tin phù hợp với đơn hàng, mã xác minh sẽ được gửi đến thông tin liên hệ đó.',
    );
  }

  @override
  Future<GuestOrderAccess> verify(
    String challengeId,
    String otp, {
    required String idempotencyKey,
  }) async {
    await Future<void>.delayed(const Duration(milliseconds: 200));
    final challenge = _challenges[challengeId];
    if (challenge == null || !challenge.expires.isAfter(DateTime.now())) {
      throw const ApiException(
        Failure(
          kind: FailureKind.expired,
          message: 'Mã đã hết hạn. Vui lòng gửi yêu cầu mới.',
        ),
      );
    }
    if (otp != '789214' ||
        !_orders.matchesGuestContact(challenge.order, challenge.contact)) {
      throw const ApiException(
        Failure(
          kind: FailureKind.validation,
          message: 'Mã hoặc thông tin xác minh chưa hợp lệ.',
        ),
      );
    }
    _challenges.remove(challengeId);
    final expiry = DateTime.now().add(const Duration(minutes: 15));
    _orders.grantGuestAccess(challenge.order, expiry);
    return GuestOrderAccess(
      orderId: challenge.order,
      orderNumber: challenge.order,
      expiresAt: expiry,
    );
  }
}
