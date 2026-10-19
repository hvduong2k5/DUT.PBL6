import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/error/failures.dart';

class MockOrdersRepository implements OrdersRepository {
  final List<CustomerOrder> _created = [];
  void record(OrderConfirmation confirmation, CheckoutDraft draft) {
    _created.removeWhere((order) => order.id == confirmation.orderId);
    _created.add(
      CustomerOrder(
        id: confirmation.orderId,
        status: OrderStatusUi.fromApi(confirmation.status),
        rawStatus: confirmation.status,
        totalItems: draft.cart.totalItems,
        total: confirmation.total,
        createdAt: DateTime.now(),
        address: '${draft.streetAddress}, ${draft.ward}, ${draft.province}',
        items: draft.cart.items
            .map(
              (line) => OrderItem(
                name: line.product.name,
                variantName: line.variant.name,
                quantity: line.quantity,
                price: line.variant.price,
              ),
            )
            .toList(),
      ),
    );
  }

  static final _now = DateTime(2026, 9, 24, 10, 15);
  static final _orders = [
    CustomerOrder(
      id: 'OM-8921',
      status: OrderStatus.shipping,
      totalItems: 3,
      total: Money.vnd(460000),
      createdAt: _now,
      address: '12 Đường Mẫu, Phường Mẫu, Huế',
      checkpoints: [
        OrderCheckpoint(label: 'Đã đặt', time: _now),
        OrderCheckpoint(
          label: 'Đã xác nhận',
          time: _now.add(const Duration(minutes: 27)),
        ),
        OrderCheckpoint(
          label: 'Đang giao',
          time: _now.add(const Duration(hours: 3)),
        ),
      ],
      items: [
        OrderItem(
          name: 'Kẹo Mè Xửng Giòn',
          variantName: 'Túi 500g',
          quantity: 2,
          price: Money.vnd(110000),
        ),
        OrderItem(
          name: 'Kẹo Cau Huế',
          variantName: 'Túi 250g',
          quantity: 1,
          price: Money.vnd(240000),
        ),
      ],
    ),
    CustomerOrder(
      id: 'OM-8874',
      status: OrderStatus.delivered,
      totalItems: 2,
      total: Money.vnd(220000),
      createdAt: DateTime(2026, 9, 18),
      items: [
        OrderItem(
          name: 'Kẹo Mè Xửng Dẻo',
          variantName: 'Hộp 250g',
          quantity: 2,
          price: Money.vnd(110000),
        ),
      ],
    ),
    CustomerOrder(
      id: 'OM-8762',
      status: OrderStatus.cancelled,
      totalItems: 1,
      total: Money.vnd(170000),
      createdAt: DateTime(2026, 9, 10),
      items: [
        OrderItem(
          name: 'Bánh Đậu Xanh Trái Cây',
          variantName: 'Hộp 300g',
          quantity: 1,
          price: Money.vnd(170000),
        ),
      ],
    ),
  ];
  @override
  Future<List<CustomerOrder>> orders() async {
    await Future<void>.delayed(const Duration(milliseconds: 180));
    return [..._created, ..._orders];
  }

  @override
  Future<CustomerOrder> order(String id) async {
    await Future<void>.delayed(const Duration(milliseconds: 120));
    final order = [
      ..._created,
      ..._orders,
    ].where((item) => item.id == id).firstOrNull;
    if (order == null) {
      throw const ApiException(
        Failure(
          kind: FailureKind.notFound,
          message: 'Không tìm thấy đơn hàng trong phiên truy cập này.',
        ),
      );
    }
    return order;
  }
}
