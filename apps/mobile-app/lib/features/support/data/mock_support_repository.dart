import 'package:uuid/uuid.dart';
import 'package:mobile_app/core/error/failures.dart';
import 'package:mobile_app/core/network/api_exception.dart';
import 'package:mobile_app/features/auth/domain/auth_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/orders/domain/order_models.dart';
import 'package:mobile_app/features/support/domain/support_repository.dart';

class MockSupportRepository implements SupportRepository {
  MockSupportRepository({required this.currentUser, required this.orders});
  final AuthUser? Function() currentUser;
  final OrdersRepository orders;
  final Map<String, SupportTicket> _tickets = {};
  final Map<String, String> _owners = {}, _requests = {};
  final Map<String, SupportMessage> _replies = {};
  String get _owner => currentUser()?.id ?? 'local-guest-session';
  @override
  Future<SupportContext> context() async {
    final user = currentUser();
    final linked = user == null
        ? <SupportLinkedOrder>[]
        : (await orders.orders())
              .map(
                (o) => SupportLinkedOrder(
                  o.id,
                  o.id,
                  o.status.label,
                  '${o.totalItems} sản phẩm',
                ),
              )
              .toList();
    return SupportContext(
      guest: user == null,
      customerName: user?.fullName ?? '',
      customerContact: user?.phoneNumber ?? '',
      servicePromise: 'Local: phản hồi được mô phỏng, không gửi đến CSKH thật.',
      subjects: const [
        SupportOption(
          'PRODUCT_ADVICE',
          'Tư vấn thức quà',
          'Bảo quản, thưởng thức và quà biếu',
        ),
        SupportOption(
          'ORDER_HELP',
          'Hỗ trợ đơn hàng',
          'Theo dõi và giao nhận đơn đã xác minh',
        ),
        SupportOption('OTHER', 'Chủ đề khác', 'Trao đổi cùng Ban Tri Kỷ'),
      ],
      priorities: const [
        SupportOption(
          'NORMAL',
          'Bình thường',
          'Câu hỏi thông thường',
          promise: 'Theo lịch tiếp nhận',
        ),
        SupportOption(
          'IMPORTANT',
          'Ưu tiên Cố Đô',
          'Cần tư vấn sớm',
          promise: 'CSKH thẩm định ưu tiên',
        ),
      ],
      orders: linked,
    );
  }

  @override
  Future<String> create({
    required String subject,
    required String title,
    required String message,
    required String priority,
    required String orderId,
    required String idempotencyKey,
  }) async {
    final key = '$_owner:$idempotencyKey';
    if (_requests.containsKey(key)) return _requests[key]!;
    final ctx = await context();
    if (!ctx.subjects.any((s) => s.code == subject) ||
        !ctx.priorities.any((p) => p.code == priority) ||
        (orderId.isNotEmpty && !ctx.orders.any((o) => o.id == orderId))) {
      throw const ApiException(
        Failure(
          kind: FailureKind.forbidden,
          message: 'Không thể liên kết thông tin ngoài quyền truy cập.',
        ),
      );
    }
    final id = const Uuid().v4();
    _tickets[id] = SupportTicket(
      id: id,
      number: 'TK-${id.substring(0, 8).toUpperCase()}',
      title: title,
      subject: ctx.subjects.firstWhere((s) => s.code == subject).label,
      status: 'Đã tiếp nhận',
      description: 'Yêu cầu local đã được ghi nhận trong phiên thử nghiệm.',
      priority: ctx.priorities.firstWhere((p) => p.code == priority).label,
      promise: ctx.servicePromise,
      assignee: 'Đang phân công',
      canReply: true,
      order: ctx.orders.where((o) => o.id == orderId).firstOrNull,
      messages: [
        SupportMessage(
          id: const Uuid().v4(),
          role: 'CUSTOMER',
          sender: currentUser()?.fullName ?? 'Khách',
          message: message,
          time: DateTime.now(),
        ),
      ],
    );
    _owners[id] = _owner;
    _requests[key] = id;
    return id;
  }

  @override
  Future<SupportTicket> ticket(String id) async {
    if (_owners[id] != _owner || !_tickets.containsKey(id)) {
      throw const ApiException(
        Failure(
          kind: FailureKind.forbidden,
          message: 'Ticket không tồn tại hoặc bạn không có quyền xem.',
        ),
      );
    }
    return _tickets[id]!;
  }

  @override
  Future<SupportMessage> reply(
    String id,
    String message, {
    required String idempotencyKey,
  }) async {
    final detail = await ticket(id);
    if (!detail.canReply) {
      throw const ApiException(
        Failure(
          kind: FailureKind.conflict,
          message: 'Ticket đã đóng trao đổi.',
        ),
      );
    }
    final key = '$_owner:$id:$idempotencyKey';
    if (_replies.containsKey(key)) return _replies[key]!;
    final result = SupportMessage(
      id: const Uuid().v4(),
      role: 'CUSTOMER',
      sender: currentUser()?.fullName ?? 'Khách',
      message: message,
      time: DateTime.now(),
    );
    _tickets[id] = detail.withReply(result);
    _replies[key] = result;
    return result;
  }
}
