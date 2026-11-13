import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/returns/domain/returns_repository.dart';

class MockReturnsRepository implements ReturnsRepository {
  MockReturnsRepository(this._orders, this._customerId);
  final OrdersRepository _orders;
  final String? Function() _customerId;
  final Map<String, ReturnCase> _cases = {};
  final Map<String, String> _owners = {}, _requests = {};
  final Map<String, List<ReturnLine>> _lines = {};
  final Map<String, int> _claimed = {};
  String _owner() {
    final owner = _customerId();
    if (owner == null) {
      throw const ApiException(
        Failure(
          kind: FailureKind.unauthenticated,
          message: 'Đăng nhập để quản lý hồ sơ hậu mãi.',
        ),
      );
    }
    return owner;
  }

  @override
  Future<ReturnFormContext> context(String orderId) async {
    final owner = _owner(), order = await _orders.order(orderId);
    if (order.status != OrderStatus.delivered) {
      return ReturnFormContext(
        orderId: order.id,
        blockingMessage:
            'Đơn chưa đủ điều kiện tạo yêu cầu đổi trả. Vui lòng liên hệ hỗ trợ.',
      );
    }
    final key = '$owner:$orderId';
    final lines = _lines.putIfAbsent(
      key,
      () => order.items
          .map(
            (item) => ReturnLine(
              id: 'LOCAL-LINE-${const Uuid().v4()}',
              name: item.name,
              variant: item.variantName,
              maximum: item.quantity,
              price: item.price,
            ),
          )
          .toList(),
    );
    return ReturnFormContext(
      orderId: orderId,
      address: order.address,
      lines: lines
          .map(
            (line) => ReturnLine(
              id: line.id,
              name: line.name,
              variant: line.variant,
              maximum: line.maximum - (_claimed[line.id] ?? 0),
              price: line.price,
            ),
          )
          .where((l) => l.maximum > 0)
          .toList(),
    );
  }

  @override
  Future<ReturnCase> create(
    ReturnDraft draft, {
    required String idempotencyKey,
  }) async {
    final owner = _owner(), key = '$owner:$idempotencyKey';
    final existing = _requests[key];
    if (existing != null) return _cases[existing]!;
    final ctx = await context(draft.orderId);
    final line = ctx.lines.where((l) => l.id == draft.lineId).firstOrNull;
    if (line == null || draft.quantity < 1 || draft.quantity > line.maximum) {
      throw const ApiException(
        Failure(
          kind: FailureKind.conflict,
          message: 'Sản phẩm hoặc số lượng đã thay đổi. Vui lòng kiểm tra lại.',
        ),
      );
    }
    final id = 'LOCAL-RET-${const Uuid().v4()}';
    final result = ReturnCase(
      id: id,
      orderId: draft.orderId,
      status: 'REQUESTED',
      reason: draft.details,
      refund: Money.vnd(0),
      createdAt: DateTime.now(),
    );
    _cases[id] = result;
    _owners[id] = owner;
    _requests[key] = id;
    _claimed[line.id] = (_claimed[line.id] ?? 0) + draft.quantity;
    // Bank details/evidence URLs are not retained in the mock case projection.
    return result;
  }

  @override
  Future<ReturnCase> detail(String id) async {
    final owner = _owner();
    if (_owners[id] != owner || !_cases.containsKey(id)) {
      throw const ApiException(
        Failure(
          kind: FailureKind.forbidden,
          message: 'Hồ sơ không tồn tại hoặc bạn không có quyền xem.',
        ),
      );
    }
    return _cases[id]!;
  }
}
