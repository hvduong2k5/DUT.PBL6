import 'dart:convert';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';
import 'package:mobile_app/features/orders/domain/mvp_order_models.dart';
import 'package:mobile_app/features/orders/domain/mvp_orders_repository.dart';

// Process-local server fixture. No receiver profile, QR or real payment exists.
class LocalMvpOrdersRepository implements MvpOrdersRepository {
  LocalMvpOrdersRepository(this._principal);
  final String? Function() _principal;
  final _orders =
      <String, ({String owner, String intent, Map<String, dynamic> json})>{};
  final _cancelKeys = <String, ({String input, MvpCancellation result})>{};
  String _owner() =>
      _principal() ??
      (throw const ApiException(
        Failure(
          kind: FailureKind.unauthenticated,
          message: 'Vui lòng đăng nhập lại.',
        ),
      ));

  void record(
    String owner,
    String orderId,
    String operationId,
    MvpQuote quote,
  ) {
    final stamp = DateTime.now().toUtc().toIso8601String();
    _orders[orderId] = (
      owner: owner,
      intent: const Uuid().v4(),
      json: {
        'order_id': orderId,
        'operation_id': operationId,
        'order_code': 'LOCAL-${_orders.length + 1}',
        'status': 'PENDING_PAYMENT',
        'payment_status': 'UNPAID',
        'payment_method': quote.paymentMethod,
        'subtotal': quote.subtotal.units.toInt(),
        'shipping_fee': quote.shippingFee.units.toInt(),
        'final_amount': quote.total.units.toInt(),
        'version': 1,
        'operational_hold': false,
        'cancellation_status': 'NONE',
        'compensation_status': 'NOT_REQUIRED',
        'allowed_actions': ['PAY', 'CANCEL', 'CONTACT_SUPPORT'],
        'created_at': stamp,
        'updated_at': stamp,
        'shipment_id': null,
        'snapshot': {
          'quote_id': quote.id,
          'cart_revision': quote.cartRevision,
          'payment_method': quote.paymentMethod,
          'catalog_snapshot_id': quote.catalogSnapshotId,
          'shipping_quote_reference': quote.shippingQuoteReference,
          'items': quote.items
              .map(
                (i) => {
                  'sku_code': i.sku,
                  'quantity': i.quantity,
                  if (i.name != null) 'product_name': i.name,
                  if (i.unitPrice != null)
                    'unit_price': i.unitPrice!.units.toInt(),
                },
              )
              .toList(),
          'address': {
            'recipient_name': quote.address.recipient,
            'phone': quote.address.phone,
            'street': quote.address.street,
            'ward_code': quote.address.wardCode,
            'province_code': quote.address.provinceCode,
          },
          'subtotal': quote.subtotal.units.toInt(),
          'shipping_fee': quote.shippingFee.units.toInt(),
          'final_amount': quote.total.units.toInt(),
          'expires_at': quote.expiresAt.toUtc().toIso8601String(),
          'catalog_expires_at': quote.catalogExpiresAt
              .toUtc()
              .toIso8601String(),
          'shipping_expires_at': quote.shippingExpiresAt
              .toUtc()
              .toIso8601String(),
        },
      },
    );
  }

  ({String owner, String intent, Map<String, dynamic> json}) _owned(String id) {
    final owner = _owner(), record = _orders[id];
    if (record == null || record.owner != owner) {
      throw const ApiException(
        Failure(
          kind: FailureKind.notFound,
          message: 'Không tìm thấy đơn hàng.',
        ),
      );
    }
    return record;
  }

  @override
  Future<MvpOrderPageData> firstPage() async {
    final owner = _owner();
    return MvpOrderPageData.fromJson({
      'items': _orders.values
          .where((r) => r.owner == owner)
          .map((r) => r.json)
          .toList()
          .reversed
          .toList(),
      'next_cursor': '',
    });
  }

  @override
  Future<MvpOrder> order(String id) async {
    var record = _owned(id);
    if (record.json['cancellation_status'] == 'PENDING') {
      final updated = {
        ...record.json,
        'status': 'CANCELLED_BY_USER',
        'cancellation_status': 'SUCCEEDED',
        'compensation_status': 'SUCCEEDED',
        'version': (record.json['version'] as int) + 1,
        'allowed_actions': ['CONTACT_SUPPORT'],
        'updated_at': DateTime.now().toUtc().toIso8601String(),
      };
      record = (owner: record.owner, intent: record.intent, json: updated);
      _orders[id] = record;
      for (final entry in _cancelKeys.entries.toList()) {
        if (entry.value.result.orderId == id) {
          _cancelKeys[entry.key] = (
            input: entry.value.input,
            result: MvpCancellation.fromJson({
              ...entry.value.result.toJson(),
              'status': 'SUCCEEDED',
              'compensation_status': 'SUCCEEDED',
              'version': updated['version'],
            }),
          );
        }
      }
    }
    return MvpOrder.fromJson(record.json);
  }

  @override
  Future<MvpPaymentStatus> payment(String id) async {
    final record = _owned(id), order = MvpOrder.fromJson(record.json);
    final cancelled = order.status.startsWith('CANCELLED_');
    final now = DateTime.now().toUtc();
    return MvpPaymentStatus.fromJson({
      'order_id': id,
      'intent_id': record.intent,
      'provider': 'SEPAY',
      'method': 'VIETQR',
      'currency': 'VND',
      'order_status': order.status,
      'payment_status': order.paymentStatus,
      'display_state': cancelled ? 'CANCELLED' : 'WAITING_PAYMENT',
      'intent_status': cancelled ? 'CANCELLED' : 'ACTIVE',
      'expected_amount': order.total.units.toInt(),
      'allocated_amount': 0,
      'refunded_amount': 0,
      'has_open_reconciliation': false,
      'can_pay': false,
      'version': order.version,
      'next_poll_after_seconds': 0,
      'server_time': now.toIso8601String(),
      'expires_at': DateTime.parse(
        record.json['created_at'] as String,
      ).add(const Duration(minutes: 10)).toIso8601String(),
      'verification_due_at': null,
      'verification_started_at': null,
      'support_reference': null,
      'support_reason': null,
    });
  }

  @override
  Future<MvpPaymentInstructions> instructions(String id) async {
    _owned(id);
    throw const ApiException(
      Failure(
        kind: FailureKind.validation,
        message: 'Local chưa có hướng dẫn chuyển khoản thật.',
      ),
    );
  }

  @override
  Future<MvpCancellation> cancel(MvpCancellationCommand command) async {
    final record = _owned(command.orderId);
    final scopedKey = jsonEncode([record.owner, command.key]);
    final body = jsonEncode([command.orderId, command.body, command.version]);
    final previous = _cancelKeys[scopedKey];
    if (previous != null) {
      if (previous.input != body) {
        throw const ApiException(
          Failure(
            kind: FailureKind.conflict,
            message: 'Yêu cầu hủy không khớp.',
          ),
        );
      }
      return previous.result;
    }
    final order = MvpOrder.fromJson(record.json);
    if (order.version != command.version || !order.actions.contains('CANCEL')) {
      throw const ApiException(
        Failure(
          kind: FailureKind.conflict,
          message: 'Đơn đã thay đổi. Kiểm tra lại yêu cầu hủy.',
        ),
      );
    }
    final result = MvpCancellation.fromJson({
      'cancellation_id': const Uuid().v4(),
      'order_id': command.orderId,
      'status': 'PENDING',
      'compensation_status': 'PENDING',
      'version': order.version + 1,
    });
    _cancelKeys[scopedKey] = (input: body, result: result);
    _orders[command.orderId] = (
      owner: record.owner,
      intent: record.intent,
      json: {
        ...record.json,
        'version': order.version + 1,
        'cancellation_status': 'PENDING',
        'compensation_status': 'PENDING',
        'allowed_actions': ['CONTACT_SUPPORT'],
      },
    );
    return result;
  }
}
