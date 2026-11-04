import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_text_field.dart';
import 'package:mobile_app/core/widgets/public_https_button.dart';
import 'package:mobile_app/features/auth/presentation/auth_return_location.dart';
import 'package:mobile_app/features/auth/presentation/capabilities_cubit.dart';
import 'package:mobile_app/features/catalog/domain/catalog_models.dart';
import 'package:mobile_app/features/orders/domain/orders_repository.dart';
import 'package:mobile_app/features/reviews/domain/reviews_repository.dart';
import 'package:mobile_app/features/reviews/presentation/reviews_cubits.dart';

class ProductReviewsSection extends StatelessWidget {
  const ProductReviewsSection({super.key, required this.product});
  final Product product;
  Future<void> _write(BuildContext context) async {
    final caps = context.read<CapabilitiesCubit>();
    final created = await showModalBottomSheet<ProductReview>(
      context: context,
      isScrollControlled: true,
      isDismissible: false,
      enableDrag: false,
      useSafeArea: true,
      showDragHandle: true,
      builder: (_) => BlocProvider(
        create: (_) => ReviewFormCubit(
          context.read<ReviewsRepository>(),
          context.read<OrdersRepository>(),
          product,
          mayCreate: () => caps.state.allows('REVIEW_CREATE'),
        )..loadOrders(),
        child: _ReviewForm(product: product),
      ),
    );
    if (created != null && context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Đánh giá đã được ghi nhận. Cảm ơn bạn đã chia sẻ.'),
        ),
      );
      context.read<ReviewsCubit>().load();
    }
  }

  @override
  Widget build(
    BuildContext context,
  ) => BlocBuilder<CapabilitiesCubit, CapabilitiesState>(
    builder: (context, caps) => BlocBuilder<ReviewsCubit, ReviewsState>(
      builder: (context, state) {
        final data = state.data, cubit = context.read<ReviewsCubit>();
        return Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Divider(height: 40),
            Text(
              'Tri kỷ nói gì về thức quà này?',
              style: Theme.of(context).textTheme.titleLarge,
            ),
            const SizedBox(height: 12),
            if (caps.allows('REVIEW_CREATE'))
              OutlinedButton(
                onPressed: () => _write(context),
                child: const Text('Viết đánh giá'),
              )
            else if (!caps.loading && caps.projection?.actor == 'GUEST')
              TextButton(
                onPressed: () => context.push(
                  authLocation(
                    AppRoutes.login,
                    '/product/${Uri.encodeComponent(product.id)}',
                  ),
                ),
                child: const Text('Đăng nhập để đánh giá'),
              ),
            if (state.loading)
              const Padding(
                padding: EdgeInsets.all(24),
                child: Center(child: CircularProgressIndicator()),
              ),
            if (state.error != null) ...[
              Text(state.error!),
              TextButton(
                onPressed: cubit.load,
                child: const Text('Thử tải đánh giá lại'),
              ),
            ],
            if (data != null) ...[
              if (data.total > 0)
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        '${NumberFormat('0.#', 'vi').format(data.average)} trên 5 · ${data.total} đánh giá',
                      ),
                      _ReviewStars(data.average),
                    ],
                  ),
                ),
              if (data.reviews.isEmpty)
                const Padding(
                  padding: EdgeInsets.symmetric(vertical: 20),
                  child: Text(
                    'Chưa có đánh giá. Hãy chia sẻ trải nghiệm sau khi nhận hàng.',
                  ),
                ),
              for (final review in data.reviews)
                Card(
                  margin: const EdgeInsets.symmetric(vertical: 8),
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          review.customer,
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        _ReviewStars(review.rating.toDouble()),
                        Text(
                          DateFormat(
                            'dd/MM/yyyy',
                          ).format(review.createdAt.toLocal()),
                          style: Theme.of(context).textTheme.bodySmall,
                        ),
                        if (review.verified)
                          const Padding(
                            padding: EdgeInsets.symmetric(vertical: 8),
                            child: Text(
                              'Đã mua sản phẩm',
                              style: TextStyle(color: AppColors.success),
                            ),
                          ),
                        SelectableText(review.comment),
                        for (
                          var index = 0;
                          index < review.media.length;
                          index++
                        )
                          PublicHttpsButton(
                            url: review.media[index],
                            label: 'Xem ảnh đánh giá ${index + 1}',
                          ),
                        if (review.reply.isNotEmpty)
                          Container(
                            margin: const EdgeInsets.only(top: 16),
                            padding: const EdgeInsets.all(12),
                            color: AppColors.cream,
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                const Text(
                                  'Ô Mạ phản hồi',
                                  style: TextStyle(fontWeight: FontWeight.w600),
                                ),
                                Text(review.reply),
                              ],
                            ),
                          ),
                      ],
                    ),
                  ),
                ),
              if (state.pageError != null) Text(state.pageError!),
              if (data.hasMore)
                TextButton(
                  onPressed: state.loadingMore ? null : cubit.more,
                  child: Text(
                    state.loadingMore ? 'Đang tải…' : 'Xem thêm đánh giá',
                  ),
                ),
            ],
          ],
        );
      },
    ),
  );
}

class _ReviewStars extends StatelessWidget {
  const _ReviewStars(this.value);
  final double value;
  @override
  Widget build(BuildContext context) => Semantics(
    label: '$value trên 5 sao',
    child: Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        for (var index = 0; index < 5; index++)
          Icon(
            index < value.round() ? Icons.star : Icons.star_border,
            size: 18,
            color: AppColors.honeyGold,
          ),
      ],
    ),
  );
}

class _ReviewForm extends StatefulWidget {
  const _ReviewForm({required this.product});
  final Product product;
  @override
  State<_ReviewForm> createState() => _ReviewFormState();
}

class _ReviewFormState extends State<_ReviewForm> {
  final _comment = TextEditingController(), _order = TextEditingController();
  int _rating = 5;
  @override
  void dispose() {
    _comment.dispose();
    _order.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final allowed = context.watch<CapabilitiesCubit>().state.allows(
      'REVIEW_CREATE',
    );
    return ConstrainedBox(
      constraints: BoxConstraints(
        maxHeight: MediaQuery.sizeOf(context).height * .92,
      ),
      child: Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.viewInsetsOf(context).bottom,
        ),
        child: BlocConsumer<ReviewFormCubit, ReviewFormState>(
          listener: (context, state) {
            if (state.created != null) Navigator.of(context).pop(state.created);
          },
          builder: (context, state) {
            final cubit = context.read<ReviewFormCubit>();
            return PopScope(
              canPop: !state.pending,
              child: SingleChildScrollView(
                padding: const EdgeInsets.fromLTRB(24, 8, 24, 24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Text(
                      'Chia sẻ trải nghiệm',
                      style: Theme.of(context).textTheme.headlineSmall,
                    ),
                    const SizedBox(height: 12),
                    Text(widget.product.name),
                    if (!allowed)
                      const Padding(
                        padding: EdgeInsets.all(16),
                        child: Text(
                          'Phiên hoặc quyền gửi đánh giá đã thay đổi. Vui lòng đóng biểu mẫu và đăng nhập lại.',
                        ),
                      )
                    else ...[
                      const SizedBox(height: 20),
                      const Text(
                        'Chọn đơn đã giao và đúng quy cách đã mua. Hệ thống sẽ xác minh lại khi gửi.',
                      ),
                      if (state.loading)
                        const Padding(
                          padding: EdgeInsets.all(16),
                          child: Center(child: CircularProgressIndicator()),
                        )
                      else ...[
                        const SizedBox(height: 16),
                        DropdownButtonFormField<String>(
                          key: ValueKey(
                            'order:${state.orderId}:${state.orders.length}',
                          ),
                          initialValue:
                              state.orders.any((o) => o.id == state.orderId)
                              ? state.orderId
                              : null,
                          isExpanded: true,
                          decoration: const InputDecoration(
                            labelText: 'Đơn đã giao',
                          ),
                          items: [
                            for (final order in state.orders)
                              DropdownMenuItem(
                                value: order.id,
                                child: Text(
                                  '#${order.id} · ${DateFormat('dd/MM/yyyy').format(order.createdAt)}',
                                  overflow: TextOverflow.ellipsis,
                                ),
                              ),
                          ],
                          onChanged: state.pending || state.checkingOrder
                              ? null
                              : (value) {
                                  if (value != null) cubit.selectOrder(value);
                                },
                        ),
                        if (state.orders.isEmpty)
                          const Padding(
                            padding: EdgeInsets.only(top: 8),
                            child: Text('Chưa có đơn đã giao trong danh sách.'),
                          ),
                        if (state.hasMore)
                          TextButton(
                            onPressed: state.loadingMore || state.pending
                                ? null
                                : () => cubit.loadOrders(more: true),
                            child: Text(
                              state.loadingMore
                                  ? 'Đang tải…'
                                  : 'Tải thêm đơn đã giao',
                            ),
                          ),
                        TextButton(
                          onPressed: state.pending
                              ? null
                              : () => cubit.loadOrders(),
                          child: const Text('Tải lại danh sách đơn'),
                        ),
                      ],
                      ExpansionTile(
                        title: const Text('Dùng mã đơn khác'),
                        tilePadding: EdgeInsets.zero,
                        children: [
                          AppTextField(
                            controller: _order,
                            label: 'Mã đơn cần kiểm tra',
                            enabled: !state.pending,
                            textInputAction: TextInputAction.done,
                            onSubmitted: (_) => cubit.selectOrder(_order.text),
                          ),
                          TextButton(
                            onPressed: state.pending || state.checkingOrder
                                ? null
                                : () => cubit.selectOrder(_order.text),
                            child: const Text('Kiểm tra đơn'),
                          ),
                        ],
                      ),
                      if (state.checkingOrder) const LinearProgressIndicator(),
                      if (state.skus.isNotEmpty) ...[
                        const SizedBox(height: 16),
                        DropdownButtonFormField<String>(
                          key: ValueKey(
                            'sku:${state.orderId}:${state.skus.join(',')}',
                          ),
                          initialValue: state.sku,
                          isExpanded: true,
                          decoration: const InputDecoration(
                            labelText: 'Quy cách đã mua',
                          ),
                          items: [
                            for (final sku in state.skus)
                              DropdownMenuItem(
                                value: sku,
                                child: Text(
                                  widget.product.variants
                                      .firstWhere((v) => v.sku == sku)
                                      .name,
                                ),
                              ),
                          ],
                          onChanged: state.pending
                              ? null
                              : (value) {
                                  if (value != null) cubit.selectSku(value);
                                },
                        ),
                      ],
                      const SizedBox(height: 20),
                      const Text('Mức độ hài lòng'),
                      Wrap(
                        children: [
                          for (var value = 1; value <= 5; value++)
                            IconButton(
                              tooltip: '$value sao',
                              onPressed: state.pending
                                  ? null
                                  : () => setState(() => _rating = value),
                              icon: Icon(
                                value <= _rating
                                    ? Icons.star
                                    : Icons.star_border,
                                color: AppColors.honeyGold,
                              ),
                            ),
                        ],
                      ),
                      AppTextField(
                        controller: _comment,
                        label: 'Cảm nhận của bạn',
                        maxLines: 5,
                        maxLength: 1000,
                        enabled: !state.pending,
                      ),
                      const Text(
                        'Đánh giá văn bản; hiện chưa hỗ trợ tải ảnh đính kèm.',
                      ),
                      if (state.error != null)
                        Padding(
                          padding: const EdgeInsets.symmetric(vertical: 12),
                          child: Semantics(
                            liveRegion: true,
                            child: Text(
                              state.error!,
                              style: TextStyle(
                                color: Theme.of(context).colorScheme.error,
                              ),
                            ),
                          ),
                        ),
                      const SizedBox(height: 20),
                      AppPrimaryButton(
                        label: 'Gửi đánh giá',
                        loading: state.pending,
                        onPressed: state.checkingOrder
                            ? null
                            : () {
                                FocusScope.of(context).unfocus();
                                cubit.submit(_rating, _comment.text);
                              },
                      ),
                    ],
                    TextButton(
                      onPressed: state.pending
                          ? null
                          : () => Navigator.of(context).pop(),
                      child: const Text('Đóng biểu mẫu'),
                    ),
                  ],
                ),
              ),
            );
          },
        ),
      ),
    );
  }
}
