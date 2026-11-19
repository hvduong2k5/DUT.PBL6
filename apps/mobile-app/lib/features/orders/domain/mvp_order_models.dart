import 'package:mobile_app/core/models/money.dart';
import 'package:mobile_app/features/checkout/domain/mvp_checkout_models.dart';
import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';

int mvpInt(Object? value, {int minimum = 0, int? maximum}) {
  if (value is! int || value < minimum || maximum != null && value > maximum) {
    throw const FormatException('Invalid MVP order integer');
  }
  return value;
}

bool _flag(Object? value) {
  if (value is! bool) throw const FormatException('Invalid MVP flag');
  return value;
}

Money _money(Object? value) => Money.vnd(mvpInt(value, maximum: 1000000000000));
Money _orderMoney(Object? value) => Money.vnd(mvpInt(value));
Map<String, dynamic> _map(Object? value) =>
    Map<String, dynamic>.from(value as Map);

class MvpOrder {
  MvpOrder.fromJson(Map<String, dynamic> json)
    : id = checkoutId(json['order_id']),
      operationId = checkoutId(json['operation_id']),
      code = checkoutText(json['order_code']),
      status = checkoutText(json['status']),
      paymentStatus = checkoutText(json['payment_status']),
      paymentMethod = checkoutText(json['payment_method']),
      version = mvpInt(json['version'], minimum: 1),
      total = _orderMoney(json['final_amount']),
      subtotal = _orderMoney(json['subtotal']),
      shippingFee = _orderMoney(json['shipping_fee']),
      cancellationStatus = checkoutText(json['cancellation_status']),
      compensationStatus = checkoutText(json['compensation_status']),
      hold = _flag(json['operational_hold']),
      actions = Set.unmodifiable(
        (json['allowed_actions'] as List).cast<String>(),
      ),
      createdAt = checkoutDate(json['created_at']),
      updatedAt = checkoutDate(json['updated_at']),
      snapshot = json['snapshot'] == null
          ? null
          : MvpQuote.fromJson(_map(json['snapshot'])),
      shipmentId = json['shipment_id'] as String?;
  final String id,
      operationId,
      code,
      status,
      paymentStatus,
      paymentMethod,
      cancellationStatus,
      compensationStatus;
  final int version;
  final bool hold;
  final Set<String> actions;
  final Money total, subtotal, shippingFee;
  final DateTime createdAt, updatedAt;
  final MvpQuote? snapshot;
  final String? shipmentId;
  bool get canPay =>
      actions.contains('PAY') &&
      status == 'PENDING_PAYMENT' &&
      !hold &&
      cancellationStatus == 'NONE';
  String get statusLabel => mvpOrderStatusLabel(status);
}

String mvpOrderStatusLabel(String status) =>
    const {
      'PENDING_PAYMENT': 'Chờ thanh toán',
      'PAYMENT_FINALIZING': 'Đang hoàn tất thanh toán',
      'PAID': 'Đã thanh toán',
      'CONFIRMED_COD': 'Đã xác nhận COD',
      'PROCESSING': 'Đang chuẩn bị',
      'PACKED': 'Đã đóng gói',
      'SHIPPED': 'Đang giao hàng',
      'DELIVERED': 'Đã giao hàng',
      'DELIVERY_FAILED': 'Giao hàng chưa thành công',
      'COMPLETED': 'Đã hoàn tất',
      'CANCELLED_TIMEOUT': 'Đã hủy do hết hạn',
      'CANCELLED_BY_USER': 'Đã hủy theo yêu cầu',
      'CANCELLED_BY_ADMIN': 'Đã hủy bởi cửa hàng',
    }[status] ??
    'Trạng thái đang được cập nhật';

class MvpOrderPageData {
  MvpOrderPageData.fromJson(Map<String, dynamic> json)
    : items = List.unmodifiable(
        (json['items'] as List).map((v) => MvpOrder.fromJson(_map(v))),
      ),
      nextCursor = json['next_cursor'] as String;
  final List<MvpOrder> items;
  final String nextCursor;
}

class MvpPaymentStatus {
  MvpPaymentStatus.fromJson(Map<String, dynamic> json)
    : orderId = checkoutId(json['order_id']),
      intentId = checkoutId(json['intent_id']),
      provider = checkoutText(json['provider']),
      method = checkoutText(json['method']),
      orderStatus = checkoutText(json['order_status']),
      paymentStatus = checkoutText(json['payment_status']),
      displayState = checkoutText(json['display_state']),
      intentStatus = checkoutText(json['intent_status']),
      expected = _money(json['expected_amount']),
      allocated = _money(json['allocated_amount']),
      refunded = _money(json['refunded_amount']),
      canPay = _flag(json['can_pay']),
      reconciliation = _flag(json['has_open_reconciliation']),
      version = mvpInt(json['version'], minimum: 1),
      nextPollSeconds = mvpInt(json['next_poll_after_seconds']),
      expiresAt = checkoutDate(json['expires_at']),
      serverTime = checkoutDate(json['server_time']),
      verificationDueAt = json['verification_due_at'] == null
          ? null
          : checkoutDate(json['verification_due_at']),
      supportReference = json['support_reference'] as String? {
    if (json['currency'] != 'VND' ||
        provider != 'SEPAY' ||
        method != 'VIETQR' ||
        refunded.compareTo(allocated) > 0) {
      throw const FormatException('Invalid MVP payment projection');
    }
  }
  final String orderId,
      intentId,
      provider,
      method,
      orderStatus,
      paymentStatus,
      displayState,
      intentStatus;
  final Money expected, allocated, refunded;
  final bool canPay, reconciliation;
  final int version, nextPollSeconds;
  final DateTime expiresAt, serverTime;
  final DateTime? verificationDueAt;
  final String? supportReference;
  bool payableAt(DateTime serverNow) =>
      canPay &&
      displayState == 'WAITING_PAYMENT' &&
      orderStatus == 'PENDING_PAYMENT' &&
      intentStatus == 'ACTIVE' &&
      expiresAt.difference(serverNow) > const Duration(seconds: 30);
  String get label =>
      const {
        'WAITING_PAYMENT': 'Chờ thanh toán',
        'VERIFYING': 'Đang xác minh giao dịch',
        'VERIFICATION_DELAYED': 'Xác minh đang chậm',
        'FINALIZING': 'Đang hoàn tất thanh toán',
        'PAID': 'Đã thanh toán',
        'RECONCILIATION_REQUIRED': 'Cần đối soát',
        'EXPIRED': 'Đã hết hạn thanh toán',
        'CANCELLED': 'Đã dừng thanh toán',
        'PARTIALLY_REFUNDED': 'Đã hoàn một phần',
        'REFUNDED': 'Đã hoàn tiền',
      }[displayState] ??
      'Chưa xác định trạng thái thanh toán';
}

class MvpPaymentInstructions {
  MvpPaymentInstructions.fromJson(Map<String, dynamic> json)
    : orderId = checkoutId(json['order_id']),
      intentId = checkoutId(json['intent_id']),
      amount = _money(json['amount']),
      paymentCode = checkoutText(json['payment_code']),
      transferContent = checkoutText(json['transfer_content']),
      bankName = checkoutText(_map(json['receiver'])['bank_name']),
      bankCode = checkoutText(_map(json['receiver'])['bank_code']),
      account = checkoutText(_map(json['receiver'])['account_number']),
      accountHolder = checkoutText(_map(json['receiver'])['account_holder']),
      qrUrl = Uri.parse(checkoutText(json['qr_url'])),
      expiresAt = checkoutDate(json['expires_at']),
      serverTime = checkoutDate(json['server_time']),
      version = mvpInt(json['version'], minimum: 1) {
    if (json['provider'] != 'SEPAY' ||
        json['method'] != 'VIETQR' ||
        json['currency'] != 'VND' ||
        json['can_pay'] != true ||
        amount.units <= BigInt.zero ||
        !RegExp(r'^OMP[A-Z0-9]{12}$').hasMatch(paymentCode) ||
        !const {
          'BANK_ACCOUNT',
          'OFFICIAL_VA',
        }.contains(_map(json['receiver'])['mode']) ||
        qrUrl.scheme != 'https' ||
        qrUrl.host.isEmpty ||
        qrUrl.userInfo.isNotEmpty) {
      throw const FormatException('Invalid payment instructions');
    }
  }
  final String orderId,
      intentId,
      paymentCode,
      transferContent,
      bankName,
      bankCode,
      account,
      accountHolder;
  final Money amount;
  final Uri qrUrl;
  final DateTime expiresAt, serverTime;
  final int version;
}

class MvpCancellation {
  MvpCancellation.fromJson(Map<String, dynamic> json)
    : id = checkoutId(json['cancellation_id']),
      orderId = checkoutId(json['order_id']),
      status = checkoutText(json['status']),
      compensationStatus = checkoutText(json['compensation_status']),
      version = mvpInt(json['version'], minimum: 1) {
    if (!const {
          'PENDING',
          'SUCCEEDED',
          'REJECTED',
          'MANUAL_REVIEW',
        }.contains(status) ||
        !const {
          'NOT_REQUIRED',
          'PENDING',
          'SUCCEEDED',
          'MANUAL_REVIEW',
        }.contains(compensationStatus)) {
      throw const FormatException('Invalid cancellation result');
    }
  }
  final String id, orderId, status, compensationStatus;
  final int version;
  Map<String, dynamic> toJson() => {
    'cancellation_id': id,
    'order_id': orderId,
    'status': status,
    'compensation_status': compensationStatus,
    'version': version,
  };
}

class MvpCancellationCommand {
  MvpCancellationCommand({
    required this.owner,
    required this.scope,
    required this.orderId,
    required this.key,
    required this.reason,
    required this.version,
    this.result,
  }) {
    checkoutText(owner);
    checkoutText(scope);
    checkoutId(orderId);
    if (!RegExp(r'^[!-~]{16,128}$').hasMatch(key) ||
        reason.trim().isEmpty ||
        reason.length > 255 ||
        version < 1 ||
        result != null && result!.orderId != orderId) {
      throw const FormatException('Invalid cancellation command');
    }
  }
  factory MvpCancellationCommand.fromJson(Map<String, dynamic> json) =>
      MvpCancellationCommand(
        owner: checkoutText(json['owner']),
        scope: checkoutText(json['scope']),
        orderId: checkoutId(json['order_id']),
        key: checkoutText(json['key']),
        reason: checkoutText(json['reason']),
        version: mvpInt(json['version'], minimum: 1),
        result: json['result'] == null
            ? null
            : MvpCancellation.fromJson(_map(json['result'])),
      );
  final String owner, scope, orderId, key, reason;
  final int version;
  final MvpCancellation? result;
  MvpCancellationCommand withResult(MvpCancellation value) =>
      MvpCancellationCommand(
        owner: owner,
        scope: scope,
        orderId: orderId,
        key: key,
        reason: reason,
        version: version,
        result: value,
      );
  Map<String, dynamic> get body => {'reason': reason, 'active': true};
  Map<String, dynamic> toJson() => {
    'owner': owner,
    'scope': scope,
    'order_id': orderId,
    'key': key,
    'reason': reason,
    'version': version,
    if (result != null) 'result': result!.toJson(),
  };
}
