import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';

class RestOrdersRepository implements OrdersRepository {
  const RestOrdersRepository(this._client);
  final ApiClient _client;

  @override
  Future<List<CustomerOrder>> orders() async {
    final json = await _client.getJson(
      '/api/v1/orders',
      query: const {'page': 1, 'pageSize': 50},
    );
    return (json['orders'] as List? ?? const [])
        .map((item) => _orderFromList(Map<String, dynamic>.from(item as Map)))
        .toList();
  }

  @override
  Future<CustomerOrder> order(String id) async {
    final json = await _client.getJson(
      '/api/v1/orders/${Uri.encodeComponent(id)}',
    );
    final itemValues = json['items'] as List? ?? const [];
    final totalItems = itemValues.fold<int>(0, (total, raw) {
      final item = Map<String, dynamic>.from(raw as Map);
      return total + ((item['quantity'] as num?)?.toInt() ?? 0);
    });
    final addressJson = json['shipping_address'];
    final address = addressJson is Map
        ? Address.fromApiJson(Map<String, dynamic>.from(addressJson)).formatted
        : 'Địa chỉ theo dữ liệu đơn hàng';
    return CustomerOrder(
      id: json['order_id'].toString(),
      status: OrderStatusUi.fromApi(json['status'].toString()),
      rawStatus: json['status'] as String?,
      totalItems: totalItems,
      total: Money.fromJson(
        Map<String, dynamic>.from(json['final_amount'] as Map),
      ),
      createdAt: DateTime.parse(json['created_at'].toString()),
      address: address,
      recipientName: addressJson is Map
          ? addressJson['recipient_name'] as String?
          : null,
      recipientPhone: addressJson is Map
          ? addressJson['phone_number'] as String?
          : null,
      items: itemValues.map((raw) {
        final item = raw as Map;
        return OrderItem(
          name: item['sku_code'] as String,
          variantName: 'Giá ghi trên đơn',
          sku: item['sku_code'] as String,
          quantity: item['quantity'] as int,
          price: Money.fromJson(
            Map<String, dynamic>.from(item['price'] as Map),
          ),
          unitPriceKnown: false,
        );
      }).toList(),
      subtotal: Money.fromJson(
        Map<String, dynamic>.from(json['subtotal_amount'] as Map),
      ),
      discount: Money.fromJson(
        Map<String, dynamic>.from(json['discount_amount'] as Map),
      ),
      shippingFee: Money.fromJson(
        Map<String, dynamic>.from(json['shipping_fee'] as Map),
      ),
    );
  }

  @override
  Future<OrderTracking> tracking(String id) async => OrderTracking.fromJson(
    await _client.getJson('/api/v1/orders/${Uri.encodeComponent(id)}/tracking'),
  );
  @override
  Future<String> cancel(
    String id,
    String reason, {
    required String idempotencyKey,
  }) async {
    final json = await _client.postJson(
      '/api/v1/orders/${Uri.encodeComponent(id)}/cancel',
      data: {'reason': reason},
      headers: {'Idempotency-Key': idempotencyKey},
    );
    if (json['is_cancelled'] != true) {
      throw const ApiException(
        Failure(
          kind: FailureKind.conflict,
          message: 'Đơn chưa được hủy. Vui lòng tải lại trạng thái đơn.',
        ),
      );
    }
    return json['status'] as String;
  }

  CustomerOrder _orderFromList(Map<String, dynamic> json) => CustomerOrder(
    id: json['order_id'].toString(),
    status: OrderStatusUi.fromApi(json['status'].toString()),
    rawStatus: json['status'] as String?,
    totalItems: (json['total_items'] as num).toInt(),
    total: Money.fromJson(
      Map<String, dynamic>.from(json['final_amount'] as Map),
    ),
    createdAt: DateTime.parse(json['created_at'].toString()),
  );
}
