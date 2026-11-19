class ProductReview {
  const ProductReview({
    required this.id,
    required this.customer,
    required this.rating,
    required this.comment,
    required this.media,
    required this.verified,
    required this.createdAt,
    required this.reply,
  });
  final String id, customer, comment, reply;
  final int rating;
  final List<String> media;
  final bool verified;
  final DateTime createdAt;
  factory ProductReview.fromJson(Map<String, dynamic> json) => ProductReview(
    id: json['review_id'] as String,
    customer: json['customer_name'] as String,
    rating: json['rating'] as int,
    comment: json['comment'] as String,
    media: (json['media_urls'] as List).cast<String>(),
    verified: json['is_verified_purchase'] as bool,
    createdAt: DateTime.parse(json['created_at'] as String),
    reply: json['seller_reply'] as String,
  );
}

class ProductReviewsPage {
  const ProductReviewsPage({
    required this.average,
    required this.total,
    required this.reviews,
    required this.page,
    required this.pageSize,
  });
  final double average;
  final int total, page, pageSize;
  final List<ProductReview> reviews;
  bool get hasMore => page * pageSize < total;
}

class ReviewDraft {
  const ReviewDraft({
    required this.productId,
    required this.orderId,
    required this.sku,
    required this.rating,
    required this.comment,
  });
  final String productId, orderId, sku, comment;
  final int rating;
}

abstract interface class ReviewsRepository {
  Future<ProductReviewsPage> page(
    String productId, {
    int page = 1,
    int pageSize = 10,
  });
  Future<ProductReview> create(
    ReviewDraft draft, {
    required String idempotencyKey,
  });
}
