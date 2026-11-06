import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/core/models/page_result.dart';

abstract interface class OrdersRepository {
  Future<List<CustomerOrder>> orders();
  Future<PageResult<CustomerOrder>> orderPage({
    String query = '',
    String? status,
    int page = 1,
    int pageSize = 10,
  });
  Future<CustomerOrder> order(String id);
  Future<OrderTracking> tracking(String id);
  Future<String> cancel(
    String id,
    String reason, {
    required String idempotencyKey,
  });
}
