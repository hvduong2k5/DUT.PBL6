class SupportOption {
  const SupportOption(
    this.code,
    this.label,
    this.description, {
    this.promise = '',
  });
  final String code, label, description, promise;
  factory SupportOption.fromJson(Map<String, dynamic> json) => SupportOption(
    json['code'] as String,
    json['label'] as String,
    json['description'] as String,
    promise: json['responsePromiseLabel'] as String? ?? '',
  );
}

class SupportLinkedOrder {
  const SupportLinkedOrder(this.id, this.number, this.status, this.summary);
  final String id, number, status, summary;
  factory SupportLinkedOrder.fromJson(Map<String, dynamic> json) =>
      SupportLinkedOrder(
        json['orderId'] as String,
        json['orderNumber'] as String,
        json['statusLabel'] as String,
        json['summary'] as String? ?? '',
      );
}

class SupportContext {
  const SupportContext({
    required this.guest,
    required this.customerName,
    required this.customerContact,
    required this.subjects,
    required this.priorities,
    required this.orders,
    required this.servicePromise,
  });
  final bool guest;
  final String customerName, customerContact, servicePromise;
  final List<SupportOption> subjects, priorities;
  final List<SupportLinkedOrder> orders;
  factory SupportContext.fromJson(Map<String, dynamic> json) {
    final customer = json['customer'] as Map?;
    return SupportContext(
      guest: json['customerMode'] == 'GUEST',
      customerName: customer?['displayName'] as String? ?? '',
      customerContact: [
        customer?['phoneDisplay'],
        customer?['emailDisplay'],
      ].whereType<String>().where((s) => s.isNotEmpty).join(' · '),
      subjects: (json['subjects'] as List)
          .map(
            (e) => SupportOption.fromJson(Map<String, dynamic>.from(e as Map)),
          )
          .toList(),
      priorities: (json['priorities'] as List)
          .map(
            (e) => SupportOption.fromJson(Map<String, dynamic>.from(e as Map)),
          )
          .toList(),
      orders: (json['linkableOrders'] as List)
          .map(
            (e) => SupportLinkedOrder.fromJson(
              Map<String, dynamic>.from(e as Map),
            ),
          )
          .toList(),
      servicePromise: json['servicePromise'] as String,
    );
  }
}

class SupportMessage {
  const SupportMessage({
    required this.id,
    required this.role,
    required this.sender,
    required this.message,
    required this.time,
    this.attachments = const [],
  });
  final String id, role, sender, message;
  final DateTime time;
  final List<String> attachments;
  factory SupportMessage.fromJson(Map<String, dynamic> json) => SupportMessage(
    id: json['messageId'] as String,
    role: json['senderRole'] as String,
    sender: json['senderDisplayName'] as String,
    message: json['message'] as String,
    time: DateTime.parse(json['occurredAt'] as String),
    attachments: (json['attachments'] as List? ?? [])
        .map((e) => '${(e as Map)['label']} · ${e['state']}')
        .toList(),
  );
}

class SupportTicket {
  const SupportTicket({
    required this.id,
    required this.number,
    required this.title,
    required this.subject,
    required this.status,
    required this.description,
    required this.priority,
    required this.promise,
    required this.assignee,
    required this.messages,
    required this.canReply,
    this.order,
  });
  final String id,
      number,
      title,
      subject,
      status,
      description,
      priority,
      promise,
      assignee;
  final List<SupportMessage> messages;
  final bool canReply;
  final SupportLinkedOrder? order;
  factory SupportTicket.fromJson(Map<String, dynamic> json) => SupportTicket(
    id: json['ticketId'] as String,
    number: json['ticketNumber'] as String,
    title: json['title'] as String,
    subject: json['subjectLabel'] as String,
    status: json['statusLabel'] as String,
    description: json['statusDescription'] as String,
    priority: json['requestedPriorityLabel'] as String,
    promise: (json['servicePromise'] as Map)['label'] as String,
    assignee:
        (json['publicAssignee'] as Map?)?['displayName'] as String? ??
        'Đang phân công',
    canReply: json['canReply'] as bool,
    order: json['linkedOrder'] is Map
        ? SupportLinkedOrder.fromJson(
            Map<String, dynamic>.from(json['linkedOrder'] as Map),
          )
        : null,
    messages: (json['messages'] as List)
        .map(
          (e) => SupportMessage.fromJson(Map<String, dynamic>.from(e as Map)),
        )
        .toList(),
  );
  SupportTicket withReply(SupportMessage value) => SupportTicket(
    id: id,
    number: number,
    title: title,
    subject: subject,
    status: status,
    description: description,
    priority: priority,
    promise: promise,
    assignee: assignee,
    canReply: canReply,
    order: order,
    messages: [...messages.where((m) => m.id != value.id), value],
  );
}

abstract interface class SupportRepository {
  Future<SupportContext> context();
  Future<String> create({
    required String subject,
    required String title,
    required String message,
    required String priority,
    required String orderId,
    required String idempotencyKey,
  });
  Future<SupportTicket> ticket(String id);
  Future<SupportMessage> reply(
    String id,
    String message, {
    required String idempotencyKey,
  });
}
