import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/returns/domain/returns_repository.dart';

class RestReturnsRepository implements ReturnsRepository {
  const RestReturnsRepository(this._client, this._orders);
  final ApiClient _client;
  final OrdersRepository _orders;
  @override
  Future<ReturnFormContext> context(String orderId) async {
    final order = await _orders.order(orderId);
    // Stable Order has no line_id. Draft eligibility does not define line items.
    // No guessed ID or undeclared endpoint can authorize a financial request.
    return ReturnFormContext(
      orderId: order.id,
      address: order.address,
      blockingMessage:
          'Chưa thể chọn sản phẩm đổi trả trong ứng dụng. Vui lòng gửi yêu cầu hỗ trợ để được kiểm tra đơn hàng.',
    );
  }

  @override
  Future<ReturnCase> create(
    ReturnDraft draft, {
    required String idempotencyKey,
  }) async => ReturnCase.fromJson(
    await _client.postJson(
      '/api/v1/returns',
      data: {
        'order_id': draft.orderId,
        'lines': [
          {'line_id': draft.lineId, 'quantity': draft.quantity},
        ],
        'reason_code': draft.reason,
        'details': draft.details,
        'evidence_media_urls': draft.evidenceUrls,
        'refund_bank_code': draft.bank,
        'refund_account_number': draft.account,
        'refund_account_holder': draft.holder,
        'idempotency_key': idempotencyKey,
      },
      headers: {'Idempotency-Key': idempotencyKey},
    ),
  );
  @override
  Future<ReturnCase> detail(String id) async => ReturnCase.fromJson(
    await _client.getJson('/api/v1/returns/${Uri.encodeComponent(id)}'),
  );
}
