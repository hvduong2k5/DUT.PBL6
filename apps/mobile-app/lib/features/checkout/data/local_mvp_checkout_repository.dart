import 'dart:convert';
import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/checkout/data/local_mvp_purchase_repository.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_repository.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';

// Process-local server fixture. It never invokes legacy checkout or payments.
class LocalMvpCheckoutRepository implements MvpCheckoutRepository {
  LocalMvpCheckoutRepository(
    this._purchase,
    this._principal, {
    DateTime Function()? now,
    this.onPlaced,
  }) : _now = now ?? DateTime.now;
  final LocalMvpPurchaseRepository _purchase;
  final String? Function() _principal;
  final DateTime Function() _now;
  final void Function(
    String owner,
    String orderId,
    String operationId,
    MvpQuote quote,
  )?
  onPlaced;
  final _records =
      <
        String,
        ({
          String owner,
          String body,
          MvpQuote quote,
          MvpCheckoutOperation operation,
          int polls,
        })
      >{};
  final _keys = <String, String>{};
  String _owner() =>
      _principal() ??
      (throw const ApiException(
        Failure(
          kind: FailureKind.unauthenticated,
          message: 'Vui lòng đăng nhập lại.',
        ),
      ));

  @override
  Future<MvpCheckoutOperation> accept(
    MvpCheckoutInput input, {
    required String key,
  }) async {
    final owner = _owner(),
        scopedKey = jsonEncode([_owner(), key]),
        body = jsonEncode(input.toJson());
    final existing = _keys[scopedKey];
    if (existing != null) {
      final record = _records[existing]!;
      if (record.body != body) {
        throw const ApiException(
          Failure(
            kind: FailureKind.conflict,
            message: 'Yêu cầu thử lại không khớp.',
          ),
        );
      }
      return record.operation;
    }
    final quote = _purchase.issuedQuote(input.quoteId);
    final cart = await _purchase.cart();
    if (_owner() != owner) throw StateError('Local identity changed');
    if (!quote.isUsableAt(_now()) ||
        quote.cartRevision != input.cartRevision ||
        cart.revision != input.cartRevision ||
        quote.paymentMethod != input.paymentMethod) {
      throw const ApiException(
        Failure(
          kind: FailureKind.conflict,
          message: 'Báo giá hoặc giỏ đã thay đổi.',
        ),
      );
    }
    final stamp = _now().toUtc().toIso8601String();
    final operation = MvpCheckoutOperation.fromJson({
      'operation_id': const Uuid().v4(),
      'status': 'ACCEPTED',
      'stage': 'VALIDATING',
      'order_id': null,
      'accepted_at': stamp,
      'updated_at': stamp,
    });
    _records[operation.id] = (
      owner: owner,
      body: body,
      quote: quote,
      operation: operation,
      polls: 0,
    );
    _keys[scopedKey] = operation.id;
    return operation;
  }

  @override
  Future<MvpCheckoutOperation> operation(String id) async {
    final owner = _owner(), record = _records[id];
    if (record == null || record.owner != owner) {
      throw const ApiException(
        Failure(kind: FailureKind.notFound, message: 'Không tìm thấy yêu cầu.'),
      );
    }
    if (record.operation.terminal) return record.operation;
    final polls = record.polls + 1;
    final completed = polls >= 2;
    final operation = MvpCheckoutOperation.fromJson({
      ...record.operation.toJson(),
      'status': completed ? 'SUCCEEDED' : 'PROCESSING',
      'stage': completed ? 'DONE' : 'ACQUIRING',
      'order_id': completed ? const Uuid().v4() : null,
      'updated_at': _now().toUtc().toIso8601String(),
    });
    _records[id] = (
      owner: record.owner,
      body: record.body,
      quote: record.quote,
      operation: operation,
      polls: polls,
    );
    if (completed) {
      onPlaced?.call(owner, operation.orderId!, operation.id, record.quote);
    }
    return operation;
  }
}
