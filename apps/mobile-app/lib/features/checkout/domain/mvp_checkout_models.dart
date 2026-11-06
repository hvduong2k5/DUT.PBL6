import 'package:mobile_app/features/checkout/domain/mvp_purchase_models.dart';

String checkoutText(Object? value) {
  if (value is! String || value.trim().isEmpty) {
    throw const FormatException('Missing checkout field');
  }
  return value;
}

String checkoutId(Object? value) {
  final id = checkoutText(value);
  if (!RegExp(
    r'^[0-9a-fA-F]{8}(-[0-9a-fA-F]{4}){3}-[0-9a-fA-F]{12}$',
  ).hasMatch(id)) {
    throw const FormatException('Invalid checkout UUID');
  }
  return id;
}

DateTime checkoutDate(Object? value) {
  final text = checkoutText(value);
  if (!RegExp(r'(Z|[+-]\d{2}:\d{2})$').hasMatch(text)) {
    throw const FormatException('Checkout timestamp requires timezone');
  }
  return DateTime.parse(text);
}

class MvpCheckoutOperation {
  MvpCheckoutOperation.fromJson(Map<String, dynamic> json)
    : id = checkoutId(json['operation_id']),
      status = checkoutText(json['status']),
      stage = checkoutText(json['stage']),
      orderId = json['order_id'] == null ? null : checkoutId(json['order_id']),
      errorCode = json['error_code'] as String?,
      acceptedAt = checkoutDate(json['accepted_at']),
      updatedAt = checkoutDate(json['updated_at']),
      retryAt = json['retry_at'] == null
          ? null
          : checkoutDate(json['retry_at']) {
    if (!json.containsKey('order_id') ||
        !const {
          'ACCEPTED',
          'PROCESSING',
          'WAITING_RETRY',
          'COMPENSATING',
          'SUCCEEDED',
          'FAILED',
          'MANUAL_REVIEW',
        }.contains(status) ||
        !const {
          'VALIDATING',
          'ACQUIRING',
          'PLACING',
          'COMPENSATING',
          'DONE',
        }.contains(stage) ||
        (status == 'SUCCEEDED') != (orderId != null) ||
        status == 'FAILED' &&
            (errorCode == null || errorCode!.trim().isEmpty) ||
        updatedAt.isBefore(acceptedAt)) {
      throw const FormatException('Invalid checkout operation');
    }
  }
  final String id, status, stage;
  final String? orderId, errorCode;
  final DateTime acceptedAt, updatedAt;
  final DateTime? retryAt;
  bool get terminal => status == 'SUCCEEDED' || status == 'FAILED';
  bool get automaticPoll => !terminal && status != 'MANUAL_REVIEW';
  Map<String, dynamic> toJson() => {
    'operation_id': id,
    'status': status,
    'stage': stage,
    'order_id': orderId,
    if (errorCode != null) 'error_code': errorCode,
    'accepted_at': acceptedAt.toUtc().toIso8601String(),
    'updated_at': updatedAt.toUtc().toIso8601String(),
    if (retryAt != null) 'retry_at': retryAt!.toUtc().toIso8601String(),
  };
}

class MvpCheckoutInput {
  MvpCheckoutInput({
    required this.quoteId,
    required this.cartRevision,
    required this.paymentMethod,
  }) {
    checkoutId(quoteId);
    if (cartRevision < 0 || !const {'VIETQR', 'COD'}.contains(paymentMethod)) {
      throw const FormatException('Invalid checkout input');
    }
  }
  factory MvpCheckoutInput.fromQuote(MvpQuote quote) => MvpCheckoutInput(
    quoteId: quote.id,
    cartRevision: quote.cartRevision,
    paymentMethod: quote.paymentMethod,
  );
  factory MvpCheckoutInput.fromJson(Map<String, dynamic> json) {
    if (json['cart_revision'] is! int) {
      throw const FormatException('Invalid cart revision');
    }
    return MvpCheckoutInput(
      quoteId: checkoutId(json['quote_id']),
      cartRevision: json['cart_revision'] as int,
      paymentMethod: checkoutText(json['payment_method']),
    );
  }
  final String quoteId, paymentMethod;
  final int cartRevision;
  Map<String, dynamic> toJson() => {
    'quote_id': quoteId,
    'cart_revision': cartRevision,
    'payment_method': paymentMethod,
  };
}

class MvpCheckoutAttempt {
  MvpCheckoutAttempt({
    required this.owner,
    required this.scope,
    required this.key,
    required this.input,
    this.operation,
    this.recoveryOperationId,
  }) {
    checkoutText(owner);
    checkoutText(scope);
    if (recoveryOperationId != null) checkoutId(recoveryOperationId);
    if (!RegExp(r'^[!-~]{16,128}$').hasMatch(key)) {
      throw const FormatException('Invalid checkout key');
    }
  }
  factory MvpCheckoutAttempt.fromJson(Map<String, dynamic> json) =>
      MvpCheckoutAttempt(
        owner: checkoutText(json['owner']),
        scope: checkoutText(json['scope']),
        key: checkoutText(json['key']),
        input: MvpCheckoutInput.fromJson(
          Map<String, dynamic>.from(json['input'] as Map),
        ),
        operation: json['operation'] == null
            ? null
            : MvpCheckoutOperation.fromJson(
                Map<String, dynamic>.from(json['operation'] as Map),
              ),
        recoveryOperationId: json['recovery_operation_id'] as String?,
      );
  final String owner, scope, key;
  final MvpCheckoutInput input;
  final MvpCheckoutOperation? operation;
  final String? recoveryOperationId;
  String? get pollId => operation?.id ?? recoveryOperationId;
  MvpCheckoutAttempt withOperation(MvpCheckoutOperation value) =>
      MvpCheckoutAttempt(
        owner: owner,
        scope: scope,
        key: key,
        input: input,
        operation: value,
      );
  MvpCheckoutAttempt withRecoveryId(String value) => MvpCheckoutAttempt(
    owner: owner,
    scope: scope,
    key: key,
    input: input,
    recoveryOperationId: checkoutId(value),
  );
  Map<String, dynamic> toJson() => {
    'owner': owner,
    'scope': scope,
    'key': key,
    'input': input.toJson(),
    if (operation != null) 'operation': operation!.toJson(),
    if (recoveryOperationId != null)
      'recovery_operation_id': recoveryOperationId,
  };
}
