import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/checkout/domain/checkout_models.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/features/auth/domain/auth_validation.dart';
import 'package:mobile_app/core/models/page_result.dart';

class MockOrdersRepository implements OrdersRepository {
  MockOrdersRepository({
    bool Function()? authenticated,
    this._principal,
  }) : _authenticated = authenticated ?? (() => true);
  final bool Function() _authenticated;
  final String? Function()? _principal;
  final _owners = <String, String?>{};
  bool _owned(String id) =>
      _principal == null ||
      (_owners.containsKey(id)
          ? _owners[id] == _principal()
          : const {
              'CUST-MOCK',
              'CUST-MOCK-B2B-VERIFIED',
            }.contains(_principal()));
  final Map<String, DateTime> _guestGrants = {};
  final Map<String, String> _contacts = {};
  final Map<String, CustomerOrder> _overrides = {};
  final Map<String, String> _cancellations = {};
  void grantGuestAccess(String id, DateTime expiry) {
    _guestGrants[id] = expiry;
  }

  bool matchesGuestContact(String id, String contact) {
    final expected =
        _contacts[id] ??
        (_orders.any((order) => order.id == id) ? '0905123456' : null);
    return expected != null &&
        AuthValidation.normalizePhone(contact).replaceFirst('+84', '0') ==
            AuthValidation.normalizePhone(expected).replaceFirst('+84', '0');
  }

  final List<CustomerOrder> _created = [];
  void recordBusinessOrder(CustomerOrder order) {
    _owners[order.id] = _principal?.call();
    _created.removeWhere((o) => o.id == order.id);
    _created.add(order);
  }

  void record(OrderConfirmation confirmation, CheckoutDraft draft) {
    _owners[confirmation.orderId] = _principal?.call();
    _contacts[confirmation.orderId] = draft.phone;
    grantGuestAccess(
      confirmation.orderId,
      DateTime.now().add(const Duration(minutes: 15)),
    );
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
        recipientName: draft.fullName,
        recipientPhone: draft.phone,
        items: draft.cart.items
            .map(
              (line) => OrderItem(
                name: line.product.name,
                variantName: line.variant.name,
                sku: line.variant.sku,
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
    CustomerOrder(
      id: 'OM-8860',
      status: OrderStatus.delivered,
      rawStatus: 'DELIVERED',
      totalItems: 1,
      total: Money.vnd(85000),
      createdAt: DateTime(2026, 9, 17),
      items: [
        OrderItem(
          name: 'Mè Xửng Giòn Ngự Tiến',
          variantName: 'Hộp 500g',
          sku: 'MX-GION-500G',
          quantity: 1,
          price: Money.vnd(85000),
        ),
      ],
    ),
  ];
  @override
  Future<List<CustomerOrder>> orders() async {
    await Future<void>.delayed(const Duration(milliseconds: 180));
    if (!_authenticated()) {
      throw const ApiException(
        Failure(
          kind: FailureKind.unauthenticated,
          message: 'Vui lòng đăng nhập để xem lịch sử đơn hàng.',
        ),
      );
    }
    return [
      ..._created,
      ..._orders,
    ].where((o) => _owned(o.id)).map((o) => _overrides[o.id] ?? o).toList();
  }

  @override
  Future<CustomerOrder> order(String id) async {
    await Future<void>.delayed(const Duration(milliseconds: 120));
    final granted =
        (_principal == null || _principal() == null) &&
        (_guestGrants[id]?.isAfter(DateTime.now()) ?? false);
    if (!_authenticated() && !granted) {
      throw const ApiException(
        Failure(
          kind: FailureKind.unauthenticated,
          message: 'Vui lòng đăng nhập hoặc xác minh quyền sở hữu đơn hàng.',
        ),
      );
    }
    final order = [
      ..._created,
      ..._orders,
    ].where((item) => item.id == id).firstOrNull;
    if (order == null || !_owned(id) && !granted) {
      throw const ApiException(
        Failure(
          kind: FailureKind.notFound,
          message: 'Không tìm thấy đơn hàng trong phiên truy cập này.',
        ),
      );
    }
    return _overrides[id] ?? order;
  }

  @override
  Future<PageResult<CustomerOrder>> orderPage({
    String query = '',
    String? status,
    int page = 1,
    int pageSize = 10,
  }) async {
    final values =
        (await orders())
            .where(
              (o) =>
                  (status == null ||
                      o.status == OrderStatusUi.fromApi(status)) &&
                  (query.isEmpty ||
                      o.id.toLowerCase().contains(query.toLowerCase()) ||
                      o.items.any(
                        (i) =>
                            i.name.toLowerCase().contains(query.toLowerCase()),
                      )),
            )
            .toList()
          ..sort((a, b) => b.createdAt.compareTo(a.createdAt));
    return PageResult(
      items: values.skip((page - 1) * pageSize).take(pageSize).toList(),
      total: values.length,
      page: page,
      pageSize: pageSize,
    );
  }

  @override
  Future<OrderTracking> tracking(String id) async {
    final value = await order(id);
    return OrderTracking(
      orderId: id,
      code: value.status == OrderStatus.shipping ? 'LOCAL-$id' : '',
      carrier: value.status == OrderStatus.shipping
          ? 'Đơn vị giao nhận local'
          : '',
      status: value.rawStatus ?? value.status.label,
      seal: '',
      video: '',
      shipper: '',
      phone: '',
      checkpoints: value.checkpoints
          .map(
            (p) => ShippingCheckpoint(
              status: p.label,
              time: p.time,
              location: 'Dữ liệu local',
              description: p.label,
            ),
          )
          .toList(),
    );
  }

  @override
  Future<String> cancel(
    String id,
    String reason, {
    required String idempotencyKey,
  }) async {
    final value = await order(id);
    if (!_authenticated()) {
      throw const ApiException(
        Failure(
          kind: FailureKind.forbidden,
          message: 'Đơn khách hiện chỉ được xem. Vui lòng liên hệ hỗ trợ.',
        ),
      );
    }
    final key = '$id:$idempotencyKey';
    if (_cancellations.containsKey(key)) return _cancellations[key]!;
    if (!const {
          'PENDING',
          'PENDING_PAYMENT',
          'AWAITING_CONFIRMATION',
        }.contains(value.rawStatus) ||
        reason.trim().isEmpty) {
      throw const ApiException(
        Failure(
          kind: FailureKind.conflict,
          message:
              'Đơn không còn đủ điều kiện hủy. Vui lòng kiểm tra trạng thái mới nhất.',
        ),
      );
    }
    _overrides[id] = value.withStatus('CANCELLED_BY_USER');
    _cancellations[key] = 'CANCELLED_BY_USER';
    return 'CANCELLED_BY_USER';
  }
}
