import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/support/domain/support_repository.dart';

class RestSupportRepository implements SupportRepository {
  const RestSupportRepository(this._client);
  final ApiClient _client;
  @override
  Future<SupportContext> context() async =>
      SupportContext.fromJson(await _client.getJson('/api/v1/support/context'));
  @override
  Future<String> create({
    required String subject,
    required String title,
    required String message,
    required String priority,
    required String orderId,
    required String idempotencyKey,
  }) async {
    final json = await _client.postJson(
      '/api/v1/support/tickets',
      data: {
        'subjectCode': subject,
        'title': title,
        'message': message,
        'requestedPriority': priority,
        'linkedOrderId': orderId,
        'attachments': <Object>[],
        'idempotencyKey': idempotencyKey,
      },
      headers: {'Idempotency-Key': idempotencyKey},
    );
    return json['ticketId'] as String;
  }

  @override
  Future<SupportTicket> ticket(String id) async => SupportTicket.fromJson(
    await _client.getJson('/api/v1/support/tickets/${Uri.encodeComponent(id)}'),
  );
  @override
  Future<SupportMessage> reply(
    String id,
    String message, {
    required String idempotencyKey,
  }) async => SupportMessage.fromJson(
    await _client.postJson(
      '/api/v1/support/tickets/${Uri.encodeComponent(id)}/messages',
      data: {
        'message': message,
        'attachments': <Object>[],
        'idempotencyKey': idempotencyKey,
      },
      headers: {'Idempotency-Key': idempotencyKey},
    ),
  );
}
