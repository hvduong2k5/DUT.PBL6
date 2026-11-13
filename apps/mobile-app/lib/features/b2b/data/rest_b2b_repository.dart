import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/b2b/domain/b2b_repository.dart';

class RestB2bRepository implements B2bRepository {
  const RestB2bRepository(this._client);
  final ApiClient _client;
  @override
  Future<B2bContext> context() async =>
      B2bContext.fromJson(await _client.getJson('/api/v1/b2b/context'));
  @override
  Future<B2bRequestReceipt> request(
    B2bRequestDraft draft, {
    required String idempotencyKey,
  }) async => B2bRequestReceipt.fromJson(
    await _client.postJson(
      '/api/v1/b2b/quote-requests',
      data: {
        'organizationId': draft.organizationId,
        'purpose': draft.purpose,
        'requestedDeliveryDate': draft.date,
        'deliveryProvinceCode': draft.provinceCode,
        'deliveryProvinceName': draft.provinceName,
        'items': draft.quantities.entries
            .where((e) => e.value > 0)
            .map((e) => {'skuId': e.key, 'quantity': e.value})
            .toList(),
        'branding': {
          for (final key in b2bBranding.keys) key: draft.branding[key] ?? false,
          'attachments': <Object>[],
        },
        'invoiceRequested': draft.invoice,
        'sampleRequested': draft.sample,
        'idempotencyKey': idempotencyKey,
      },
      headers: {'Idempotency-Key': idempotencyKey},
    ),
  );
  @override
  Future<B2bQuote> quote(String id) async => B2bQuote.fromJson(
    await _client.getJson('/api/v1/b2b/quotes/${Uri.encodeComponent(id)}'),
  );
  @override
  Future<B2bAcceptance> accept(
    String id, {
    required String idempotencyKey,
  }) async => B2bAcceptance.fromJson(
    await _client.postJson(
      '/api/v1/b2b/quotes/${Uri.encodeComponent(id)}/accept',
      data: {'confirmation': true, 'idempotencyKey': idempotencyKey},
      headers: {'Idempotency-Key': idempotencyKey},
    ),
  );
}
