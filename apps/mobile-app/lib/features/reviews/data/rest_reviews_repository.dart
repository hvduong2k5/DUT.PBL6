import 'package:mobile_app/core/network/api_client.dart';
import 'package:mobile_app/features/reviews/domain/reviews_repository.dart';

class RestReviewsRepository implements ReviewsRepository {
  const RestReviewsRepository(this._client);
  final ApiClient _client;
  @override
  Future<ProductReviewsPage> page(
    String productId, {
    int page = 1,
    int pageSize = 10,
  }) async {
    final json = await _client.getJson(
      '/api/v1/reviews/products/${Uri.encodeComponent(productId)}',
      query: {'page': page, 'pageSize': pageSize},
    );
    return ProductReviewsPage(
      average: (json['average_rating'] as num).toDouble(),
      total: json['total_reviews'] as int,
      reviews: (json['reviews'] as List)
          .map(
            (e) => ProductReview.fromJson(Map<String, dynamic>.from(e as Map)),
          )
          .toList(),
      page: page,
      pageSize: pageSize,
    );
  }

  @override
  Future<ProductReview> create(
    ReviewDraft draft, {
    required String idempotencyKey,
  }) async => ProductReview.fromJson(
    await _client.postJson(
      '/api/v1/reviews',
      data: {
        'order_id': draft.orderId,
        'sku_code': draft.sku,
        'rating': draft.rating,
        'comment': draft.comment,
        'media_urls': <Object>[],
      },
      headers: {'Idempotency-Key': idempotencyKey},
    ),
  );
}
