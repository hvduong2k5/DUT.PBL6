import 'package:mobile_app/features/orders/domain/order_models.dart';

abstract interface class OrdersRepository {
  Future<List<CustomerOrder>> orders();
  Future<CustomerOrder> order(String id);
}
