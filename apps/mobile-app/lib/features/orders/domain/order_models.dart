import 'package:equatable/equatable.dart';
import 'package:mobile_app/core/models/money.dart';

enum OrderStatus { pending, confirmed, shipping, delivered, cancelled }

extension OrderStatusUi on OrderStatus {
  String get label => switch (this) {
    OrderStatus.pending => 'Chờ xác nhận',
    OrderStatus.confirmed => 'Đã xác nhận',
    OrderStatus.shipping => 'Đang giao',
    OrderStatus.delivered => 'Đã giao',
    OrderStatus.cancelled => 'Đã hủy',
  };
  static OrderStatus fromApi(String value) => switch (value.toUpperCase()) {
    'PENDING' ||
    'PENDING_PAYMENT' ||
    'AWAITING_CONFIRMATION' => OrderStatus.pending,
    'CONFIRMED' ||
    'PAYMENT_CONFIRMED' ||
    'PAID' ||
    'PROCESSING' ||
    'PACKED' => OrderStatus.confirmed,
    'SHIPPING' || 'IN_TRANSIT' => OrderStatus.shipping,
    'DELIVERED' || 'COMPLETED' => OrderStatus.delivered,
    'CANCELLED' || 'CANCELLED_BY_USER' => OrderStatus.cancelled,
    _ => OrderStatus.pending,
  };
}

class CustomerOrder extends Equatable {
  const CustomerOrder({
    required this.id,
    required this.status,
    required this.totalItems,
    required this.total,
    required this.createdAt,
    this.address = 'Địa chỉ theo dữ liệu đơn hàng',
    this.checkpoints = const [],
    this.items = const [],
  });
  final String id;
  final OrderStatus status;
  final int totalItems;
  final Money total;
  final DateTime createdAt;
  final String address;
  final List<OrderCheckpoint> checkpoints;
  final List<OrderItem> items;
  @override
  List<Object?> get props => [
    id,
    status,
    totalItems,
    total,
    createdAt,
    address,
    checkpoints,
    items,
  ];
}

class OrderItem extends Equatable {
  const OrderItem({
    required this.name,
    required this.variantName,
    required this.quantity,
    required this.price,
  });
  final String name;
  final String variantName;
  final int quantity;
  // The OpenAPI field is named `price` without defining unit-vs-line semantics.
  // Local checkout and mock data use it as unit price; REST items remain unmapped.
  final Money price;

  Money get lineTotal => price.times(quantity);

  @override
  List<Object?> get props => [name, variantName, quantity, price];
}

class OrderCheckpoint extends Equatable {
  const OrderCheckpoint({
    required this.label,
    required this.time,
    this.completed = true,
  });
  final String label;
  final DateTime time;
  final bool completed;
  @override
  List<Object?> get props => [label, time, completed];
}
