import 'package:equatable/equatable.dart';
import 'package:mobile_app/core/models/money.dart';

enum OrderStatus { pending, confirmed, shipping, delivered, cancelled, unknown }

extension OrderStatusUi on OrderStatus {
  String get label => switch (this) {
    OrderStatus.pending => 'Chờ xác nhận',
    OrderStatus.confirmed => 'Đã xác nhận',
    OrderStatus.shipping => 'Đang giao',
    OrderStatus.delivered => 'Đã giao',
    OrderStatus.cancelled => 'Đã hủy',
    OrderStatus.unknown => 'Trạng thái khác',
  };
  static OrderStatus fromApi(String value) => switch (value.toUpperCase()) {
    'PENDING' ||
    'PENDING_PAYMENT' ||
    'AWAITING_CONFIRMATION' => OrderStatus.pending,
    'CONFIRMED' ||
    'PAYMENT_CONFIRMED' ||
    'PAID' ||
    'PROCESSING' ||
    'PACKED' ||
    'PACKING' => OrderStatus.confirmed,
    'SHIPPING' || 'SHIPPED' || 'IN_TRANSIT' => OrderStatus.shipping,
    'DELIVERED' || 'COMPLETED' => OrderStatus.delivered,
    'CANCELLED' || 'CANCELLED_BY_USER' => OrderStatus.cancelled,
    _ => OrderStatus.unknown,
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
    this.rawStatus,
    this.subtotal,
    this.discount,
    this.shippingFee,
    this.recipientName,
    this.recipientPhone,
  });
  final String id;
  final OrderStatus status;
  final int totalItems;
  final Money total;
  final DateTime createdAt;
  final String address;
  final List<OrderCheckpoint> checkpoints;
  final List<OrderItem> items;
  final String? rawStatus;
  final Money? subtotal, discount, shippingFee;
  final String? recipientName, recipientPhone;
  CustomerOrder withStatus(String value) => CustomerOrder(
    id: id,
    status: OrderStatusUi.fromApi(value),
    totalItems: totalItems,
    total: total,
    createdAt: createdAt,
    address: address,
    checkpoints: checkpoints,
    items: items,
    rawStatus: value,
    subtotal: subtotal,
    discount: discount,
    shippingFee: shippingFee,
    recipientName: recipientName,
    recipientPhone: recipientPhone,
  );
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
    rawStatus,
    subtotal,
    discount,
    shippingFee,
    recipientName,
    recipientPhone,
  ];
}

class OrderItem extends Equatable {
  const OrderItem({
    required this.name,
    required this.variantName,
    required this.quantity,
    required this.price,
    this.sku,
    this.unitPriceKnown = true,
  });
  final String name;
  final String variantName;
  final int quantity;
  final String? sku;
  final bool unitPriceKnown;
  // The OpenAPI field is named `price` without defining unit-vs-line semantics.
  // Mock uses unit price; REST preserves raw price with unitPriceKnown=false.
  final Money price;

  Money get lineTotal => price.times(quantity);

  @override
  List<Object?> get props => [
    name,
    variantName,
    quantity,
    price,
    sku,
    unitPriceKnown,
  ];
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

class ShippingCheckpoint {
  const ShippingCheckpoint({
    required this.status,
    required this.time,
    required this.location,
    required this.description,
  });
  final String status, location, description;
  final DateTime time;
}

class OrderTracking {
  const OrderTracking({
    required this.orderId,
    required this.code,
    required this.carrier,
    required this.status,
    required this.seal,
    required this.video,
    required this.shipper,
    required this.phone,
    required this.checkpoints,
  });
  final String orderId, code, carrier, status, seal, video, shipper, phone;
  final List<ShippingCheckpoint> checkpoints;
  factory OrderTracking.fromJson(Map<String, dynamic> json) => OrderTracking(
    orderId: json['order_id'] as String,
    code: json['tracking_code'] as String,
    carrier: json['carrier_name'] as String,
    status: json['current_status'] as String,
    seal: json['seal_code'] as String,
    video: json['packing_video_url'] as String,
    shipper: json['shipper_name'] as String,
    phone: json['shipper_phone'] as String,
    checkpoints: (json['checkpoints'] as List).map((raw) {
      final item = raw as Map;
      return ShippingCheckpoint(
        status: item['status'] as String,
        time: DateTime.parse(item['timestamp'] as String),
        location: item['location'] as String,
        description: item['description'] as String,
      );
    }).toList(),
  );
}
