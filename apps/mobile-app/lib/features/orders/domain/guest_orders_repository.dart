class GuestOrderChallenge {
  const GuestOrderChallenge({
    required this.id,
    required this.maskedDestination,
    required this.expiresAt,
    required this.resendAfterSeconds,
    required this.message,
  });
  final String id;
  final String maskedDestination;
  final DateTime expiresAt;
  final int resendAfterSeconds;
  final String message;
}

class GuestOrderAccess {
  const GuestOrderAccess({
    required this.orderId,
    required this.orderNumber,
    required this.expiresAt,
  });
  final String orderId;
  final String orderNumber;
  final DateTime expiresAt;
}

abstract interface class GuestOrdersRepository {
  Future<GuestOrderChallenge> challenge(
    String orderNumber,
    String contact, {
    required String idempotencyKey,
  });
  Future<GuestOrderAccess> verify(
    String challengeId,
    String otp, {
    required String idempotencyKey,
  });
}
