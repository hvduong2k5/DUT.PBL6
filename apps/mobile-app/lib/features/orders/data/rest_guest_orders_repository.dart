import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/orders/domain/guest_orders_repository.dart';

class RestGuestOrdersRepository implements GuestOrdersRepository {
  const RestGuestOrdersRepository(this._client);
  final ApiClient _client;
  @override
  Future<GuestOrderChallenge> challenge(
    String orderNumber,
    String contact, {
    required String idempotencyKey,
  }) async {
    final json = await _client.postJson(
      '/api/v1/orders/guest-access/challenges',
      data: {'orderNumber': orderNumber, 'contact': contact},
      headers: {'Idempotency-Key': idempotencyKey},
    );
    return GuestOrderChallenge(
      id: json['challengeId'] as String,
      maskedDestination: json['maskedDestination'] as String,
      expiresAt: DateTime.parse(json['expiresAt'] as String),
      resendAfterSeconds: json['resendAfterSeconds'] as int,
      message: json['message'] as String,
    );
  }

  @override
  Future<GuestOrderAccess> verify(
    String challengeId,
    String otp, {
    required String idempotencyKey,
  }) async {
    final json = await _client.postJson(
      '/api/v1/orders/guest-access/challenges/${Uri.encodeComponent(challengeId)}/verify',
      data: {'otp': otp},
      headers: {'Idempotency-Key': idempotencyKey},
    );
    // Navigation uses the native orderId route, never an arbitrary returned URL.
    return GuestOrderAccess(
      orderId: json['orderId'] as String,
      orderNumber: json['orderNumber'] as String,
      expiresAt: DateTime.parse(json['accessExpiresAt'] as String),
    );
  }
}
