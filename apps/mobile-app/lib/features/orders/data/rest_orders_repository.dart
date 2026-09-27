import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/account/domain/address.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';

class RestOrdersRepository implements OrdersRepository {
  const RestOrdersRepository(this._client);
  final ApiClient _client;

  @override
  Future<List<CustomerOrder>> orders() async {
    final json = await _client.getJson(
      '/api/v1/orders',
      query: const {'page': 1, 'page_size': 50},
    );
    return (json['orders'] as List? ?? const [])
        .map((item) => _orderFromList(Map<String, dynamic>.from(item as Map)))
        .toList();
  }

  @override
  Future<CustomerOrder> order(String id) async {
    final json = await _client.getJson('/api/v1/orders/$id');
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
      totalItems: totalItems,
      total: Money.fromJson(
        Map<String, dynamic>.from(json['final_amount'] as Map),
      ),
      createdAt: DateTime.parse(json['created_at'].toString()),
      address: address,
    );
  }

  CustomerOrder _orderFromList(Map<String, dynamic> json) => CustomerOrder(
    id: json['order_id'].toString(),
    status: OrderStatusUi.fromApi(json['status'].toString()),
    totalItems: (json['total_items'] as num).toInt(),
    total: Money.fromJson(
      Map<String, dynamic>.from(json['final_amount'] as Map),
    ),
    createdAt: DateTime.parse(json['created_at'].toString()),
  );
}
