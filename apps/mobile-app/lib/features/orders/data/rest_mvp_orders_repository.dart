import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';
import 'package:mobile_app/features/orders/domain/mvp_order_models.dart';
import 'package:mobile_app/features/orders/domain/mvp_orders_repository.dart';

class RestMvpOrdersRepository implements MvpOrdersRepository {
  const RestMvpOrdersRepository(this._client);
  final ApiClient _client;
  String _path(String id) => '/api/v1/orders/${checkoutId(id)}';
  @override
  Future<MvpOrderPageData> firstPage() async =>
      MvpOrderPageData.fromJson(await _client.getJson('/api/v1/orders'));
  @override
  Future<MvpOrder> order(String id) async {
    final value = MvpOrder.fromJson(await _client.getJson(_path(id)));
    if (value.id != id) throw const FormatException('Order identity mismatch');
    return value;
  }

  @override
  Future<MvpPaymentStatus> payment(String id) async {
    final value = MvpPaymentStatus.fromJson(
      await _client.getJson('${_path(id)}/payment'),
    );
    if (value.orderId != id) {
      throw const FormatException('Payment identity mismatch');
    }
    return value;
  }

  @override
  Future<MvpPaymentInstructions> instructions(String id) async {
    final value = MvpPaymentInstructions.fromJson(
      await _client.getJson('${_path(id)}/payment-instructions'),
    );
    if (value.orderId != id) {
      throw const FormatException('Payment instructions identity mismatch');
    }
    return value;
  }

  @override
  Future<MvpCancellation> cancel(MvpCancellationCommand command) async {
    final value = MvpCancellation.fromJson(
      await _client.postJson(
        '${_path(command.orderId)}/cancel',
        data: command.body,
        headers: {
          'Idempotency-Key': command.key,
          'If-Match': command.version.toString(),
        },
      ),
    );
    if (value.orderId != command.orderId || value.version < command.version) {
      throw const FormatException('Cancellation context mismatch');
    }
    return value;
  }
}
