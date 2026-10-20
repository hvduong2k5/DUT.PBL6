import 'package:mobile_app/core/models/money.dart';

const returnReasons = {
  'DAMAGED_IN_TRANSIT': 'Hư hỏng khi vận chuyển',
  'WRONG_ITEM': 'Giao sai sản phẩm',
  'QUALITY_ISSUE': 'Vấn đề chất lượng',
  'OTHER': 'Lý do khác',
};

class ReturnLine {
  const ReturnLine({
    required this.id,
    required this.name,
    required this.variant,
    required this.maximum,
    required this.price,
  });
  final String id, name, variant;
  final int maximum;
  final Money price;
}

class ReturnFormContext {
  const ReturnFormContext({
    required this.orderId,
    this.lines = const [],
    this.blockingMessage,
    this.address = '',
  });
  final String orderId, address;
  final String? blockingMessage;
  final List<ReturnLine> lines;
}

class ReturnDraft {
  const ReturnDraft({
    required this.orderId,
    required this.lineId,
    required this.quantity,
    required this.reason,
    required this.details,
    required this.evidenceUrls,
    required this.bank,
    required this.account,
    required this.holder,
  });
  final String orderId, lineId, reason, details, bank, account, holder;
  final int quantity;
  final List<String> evidenceUrls;
}

class ReturnCase {
  const ReturnCase({
    required this.id,
    required this.orderId,
    required this.status,
    required this.reason,
    required this.refund,
    required this.createdAt,
  });
  final String id, orderId, status, reason;
  final Money refund;
  final DateTime createdAt;
  String get statusLabel => switch (status) {
    'REQUESTED' || 'RETURN_REQUESTED' => 'Đã tiếp nhận yêu cầu',
    'INSPECTING' || 'REVIEWING' => 'Đang thẩm định',
    'EVIDENCE_REQUIRED' => 'Cần bổ sung bằng chứng',
    'APPROVED' => 'Đã duyệt yêu cầu',
    'REJECTED' => 'Yêu cầu bị từ chối',
    'REFUND_PENDING' => 'Đang chờ hoàn tiền',
    'REFUNDED' => 'Đã hoàn tiền',
    'CLOSED' => 'Đã đóng hồ sơ',
    _ => status,
  };
  factory ReturnCase.fromJson(Map<String, dynamic> json) => ReturnCase(
    id: json['return_id'] as String,
    orderId: json['order_id'] as String,
    status: json['status'] as String,
    reason: json['reason'] as String,
    refund: Money.fromJson(
      Map<String, dynamic>.from(json['refund_amount'] as Map),
    ),
    createdAt: DateTime.parse(json['created_at'] as String),
  );
}

abstract interface class ReturnsRepository {
  Future<ReturnFormContext> context(String orderId);
  Future<ReturnCase> create(
    ReturnDraft draft, {
    required String idempotencyKey,
  });
  Future<ReturnCase> detail(String id);
}
