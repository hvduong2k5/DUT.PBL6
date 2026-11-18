import 'package:flutter/material.dart';
import 'package:mobile_app/core/config/app_features.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/app_bottom_navigation.dart';
import 'package:mobile_app/core/widgets/app_empty_view.dart';
import 'package:mobile_app/core/widgets/app_error_view.dart';
import 'package:mobile_app/core/widgets/app_loading_view.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_search_bar.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_banner.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_button.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_frame.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_icons.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_motif.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_section_header.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_surface.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/catalog_cubit.dart';
import 'package:mobile_app/features/catalog/presentation/widgets/product_card.dart';

/// Home in the same order as the web storefront:
/// hero -> featured -> collections -> story -> trace -> account CTA.
class HomePage extends StatelessWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context) => BlocBuilder<CartCubit, CartState>(
    builder: (context, cartState) => Scaffold(
      appBar: const HeritageSearchHeader(),
      body: HeritageSurface(
        child: BlocBuilder<CatalogCubit, CatalogState>(
          builder: (context, state) => _HomeBody(state: state),
        ),
      ),
      bottomNavigationBar: AppBottomNavigation(
        active: AppNavTab.home,
        cartCount: cartState.cart.totalItems,
      ),
    ),
  );
}

const _photos = 'assets/heritage/photos';

class _HomeBody extends StatelessWidget {
  const _HomeBody({required this.state});
  final CatalogState state;

  @override
  Widget build(BuildContext context) {
    final loading =
        state.status == CatalogStatus.initial ||
        state.status == CatalogStatus.loading;
    return ListView(
      key: const Key('home-content-scroll'),
      padding: EdgeInsets.zero,
      children: [
        _Hero(onExplore: () => context.go(AppRoutes.catalog)),
        const SizedBox(height: 28),
        const _SectionPad(
          child: HeritageSectionHeader(
            kicker: 'Được yêu thích',
            title: 'Sản phẩm nổi bật',
          ),
        ),
        const SizedBox(height: 16),
        if (loading)
          const SizedBox(height: 200, child: AppLoadingView())
        else if (state.status == CatalogStatus.failure)
          SizedBox(
            height: 260,
            child: AppErrorView(
              message: state.message ?? 'Không thể tải trang chủ.',
              onRetry: context.read<CatalogCubit>().load,
            ),
          )
        else if (state.products.isEmpty)
          const SizedBox(
            height: 240,
            child: AppEmptyView(
              title: 'Chưa có sản phẩm',
              message: 'Sản phẩm sẽ hiển thị tại đây khi cửa hàng cập nhật.',
            ),
          )
        else
          _Featured(state: state),
        const SizedBox(height: 20),
        if (AppFeatures.current.enabled(AppFeature.promotions))
          const _PromotionsBlock(),
        const SizedBox(height: 28),
        const _Divider(),
        _Collections(state: state),
        const _Story(),
        if (AppFeatures.current.enabled(AppFeature.trace)) const _Trace(),
        const _AccountCta(),
        const SizedBox(height: 24),
      ],
    );
  }
}

class _SectionPad extends StatelessWidget {
  const _SectionPad({required this.child});
  final Widget child;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 16),
    child: child,
  );
}

class _Divider extends StatelessWidget {
  const _Divider();
  @override
  Widget build(BuildContext context) => Center(
    child: SvgPicture.asset(HeritageAssets.divider, width: 28, height: 18),
  );
}

class _Hero extends StatelessWidget {
  const _Hero({required this.onExplore});
  final VoidCallback onExplore;

  @override
  Widget build(BuildContext context) => HeritageSurface(
    kind: HeritageSurfaceKind.parchment,
    child: Padding(
      padding: const EdgeInsets.all(12),
      child: HeritageBanner(
        height: 340,
        slides: [
          HeritageBannerSlide(
            image: const AssetImage('$_photos/hero-studio.jpg'),
            kicker: 'Đặc sản & quà biếu Cố đô',
            title: 'Phong vị Huế',
            subtitle:
                'Mè xửng dẻo thơm, trà Huế thanh dịu, quà biếu đậm tình miền Trung.',
            actionLabel: 'Khám phá bộ sưu tập',
            onAction: onExplore,
          ),
          HeritageBannerSlide(
            image: const AssetImage('$_photos/hero-tea-table.jpg'),
            kicker: 'Bên chén trà Huế',
            title: 'Thong thả một chén trà',
            subtitle:
                'Mè xửng mạch nha cùng trà sen thanh dịu, lời mời ngồi lại bên hiên nhà rường.',
            actionLabel: 'Khám phá bộ sưu tập',
            onAction: onExplore,
          ),
          HeritageBannerSlide(
            image: const AssetImage('$_photos/sp-hop-qua-doan-vien.jpg'),
            kicker: 'Quà biếu Cố đô',
            title: 'Hộp quà trao gửi tấm lòng',
            subtitle:
                'Hộp Đoàn Viên ba khoang, đóng gói chỉn chu cho người thân và đối tác.',
            actionLabel: 'Khám phá bộ sưu tập',
            onAction: onExplore,
          ),
        ],
      ),
    ),
  );
}

class _Featured extends StatelessWidget {
  const _Featured({required this.state});
  final CatalogState state;

  @override
  Widget build(BuildContext context) {
    final featured = state.products.take(4).toList();
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Column(
        children: [
          GridView.builder(
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            itemCount: featured.length,
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 2,
              childAspectRatio: .5,
              crossAxisSpacing: 12,
              mainAxisSpacing: 12,
            ),
            itemBuilder: (context, i) {
              final product = featured[i];
              return ProductCard(
                product: product,
                onOpen: () => context.push('/product/${product.id}'),
                onAdd: () => context.push('/product/${product.id}'),
              );
            },
          ),
          const SizedBox(height: 16),
          HeritageButton(
            label: 'Xem tất cả sản phẩm',
            variant: HeritageButtonVariant.paper,
            onPressed: () => context.go(AppRoutes.catalog),
          ),
        ],
      ),
    );
  }
}

class _Collections extends StatelessWidget {
  const _Collections({required this.state});
  final CatalogState state;

  static const _images = [
    'cat-me-xung.jpg',
    'cat-tra-hue.jpg',
    'hero-tea-table.jpg',
    'cat-qua-bieu.jpg',
  ];

  @override
  Widget build(BuildContext context) {
    final categories = state.categories.where((c) => c.id != 'all').toList();
    if (categories.isEmpty) return const SizedBox(height: 28);
    return Stack(
      children: [
        const HeritageMotif(
          '05-may',
          width: 140,
          opacity: .35,
          alignment: Alignment.topRight,
        ),
        const HeritageMotif(
          '07-hat-me',
          width: 110,
          opacity: .3,
          alignment: Alignment.bottomLeft,
        ),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 28, 16, 28),
          child: Column(
            children: [
              const HeritageSectionHeader(title: 'Chọn thức quà'),
              const SizedBox(height: 16),
              for (var i = 0; i < categories.length; i++) ...[
                _CategoryCard(
                  name: categories[i].name,
                  image: '$_photos/${_images[i % _images.length]}',
                  onTap: () => context.go(
                    '${AppRoutes.catalog}?category=${Uri.encodeQueryComponent(categories[i].id)}',
                  ),
                ),
                const SizedBox(height: 12),
              ],
            ],
          ),
        ),
      ],
    );
  }
}

class _CategoryCard extends StatelessWidget {
  const _CategoryCard({
    required this.name,
    required this.image,
    required this.onTap,
  });
  final String name;
  final String image;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) => Semantics(
    button: true,
    label: name,
    child: InkWell(
      onTap: onTap,
      child: HeritageFrame(
        level: HeritageFrameLevel.card,
        padding: const EdgeInsets.all(10),
        child: Column(
          children: [
            AspectRatio(
              aspectRatio: 16 / 9,
              child: ExcludeSemantics(
                child: Image.asset(
                  image,
                  fit: BoxFit.cover,
                  width: double.infinity,
                ),
              ),
            ),
            const SizedBox(height: 10),
            Row(
              children: [
                Expanded(
                  child: Text(
                    name,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
                const Text(
                  '→',
                  style: TextStyle(fontSize: 20, color: AppColors.brandBrown),
                ),
              ],
            ),
          ],
        ),
      ),
    ),
  );
}

class _Story extends StatelessWidget {
  const _Story();

  @override
  Widget build(BuildContext context) => HeritageSurface(
    kind: HeritageSurfaceKind.parchment,
    child: Stack(
      children: [
        const HeritageMotif(
          '01-hien-nha',
          width: 150,
          opacity: .35,
          alignment: Alignment.topLeft,
        ),
        const HeritageMotif(
          '12-hoa-sen-cung-dinh-phai',
          width: 130,
          opacity: .3,
          alignment: Alignment.bottomRight,
        ),
        Padding(
          padding: const EdgeInsets.all(20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Bên chén trà Huế',
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 10),
              const Text(
                'Thưởng chén trà Huế, cảm nhận hương vị thanh dịu và nét tinh tế trong những thức quà mộc mạc của cố đô.',
              ),
              const SizedBox(height: 8),
              const Text(
                'Một miếng mè xửng dẻo thơm mùi mạch nha, nhấp cùng ngụm trà sen ấm nóng.',
                style: TextStyle(color: AppColors.textMuted),
              ),
              const SizedBox(height: 14),
              const _PlaquePhoto('story-tra-hue.jpg'),
              const SizedBox(height: 28),
              const Text(
                'DI SẢN KIẾN TRÚC & ẨM THỰC',
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 1.4,
                  color: AppColors.brandBrown,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                'Thức quà từ Huế',
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 10),
              const Text(
                'Gói Huế mang đến những thức quà mộc mạc mà tinh tế của xứ Huế: mè xửng dẻo thơm, trà Huế thanh dịu và những món quà biếu trang nhã.',
              ),
              const SizedBox(height: 14),
              const _PlaquePhoto('story-nha-ruong.jpg'),
              const SizedBox(height: 14),
              HeritageButton(
                label: 'Chọn hộp quà biếu',
                variant: HeritageButtonVariant.brown,
                onPressed: () => context.go(AppRoutes.catalog),
              ),
            ],
          ),
        ),
      ],
    ),
  );
}

class _PlaquePhoto extends StatelessWidget {
  const _PlaquePhoto(this.name);
  final String name;
  @override
  Widget build(BuildContext context) => HeritageFrame(
    level: HeritageFrameLevel.plaque,
    padding: const EdgeInsets.all(12),
    child: AspectRatio(
      aspectRatio: 900 / 620,
      child: Image.asset('$_photos/$name', fit: BoxFit.cover),
    ),
  );
}

class _Trace extends StatefulWidget {
  const _Trace();
  @override
  State<_Trace> createState() => _TraceState();
}

class _TraceState extends State<_Trace> {
  final _code = TextEditingController();

  @override
  void dispose() {
    _code.dispose();
    super.dispose();
  }

  void _submit() {
    final code = _code.text.trim();
    context.push(
      code.isEmpty ? AppRoutes.trace : '/trace/${Uri.encodeComponent(code)}',
    );
  }

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.all(12),
    child: HeritageFrame(
      level: HeritageFrameLevel.imperial,
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            '❖  HỆ THỐNG TRUY XUẤT NGUỒN GỐC',
            style: TextStyle(
              color: HeritageColors.matSang,
              fontSize: 13,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.2,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Minh bạch nguồn gốc từng hộp quà',
            style: Theme.of(context).textTheme.headlineMedium?.copyWith(
              color: HeritageColors.kem,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 8),
          const Text(
            'Mỗi hộp xuất xưởng đều gắn mã định danh. Nhập mã QR trên nhãn để xem vùng nguyên liệu, lô sản xuất và chứng nhận.',
            style: TextStyle(color: HeritageColors.kemDam, height: 1.5),
          ),
          const SizedBox(height: 16),
          const Text(
            'Mã QR trên nhãn',
            style: TextStyle(
              color: HeritageColors.kem,
              fontSize: 14,
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 6),
          TextField(
            controller: _code,
            textInputAction: TextInputAction.search,
            onSubmitted: (_) => _submit(),
            // On the imperial background the outline-notch label would sit
            // on dark brown and vanish, so the label is drawn above instead.
            decoration: const InputDecoration(
              hintText: 'Ví dụ: QR-GH-2026-LOT08',
              floatingLabelBehavior: FloatingLabelBehavior.never,
            ),
          ),
          const SizedBox(height: 12),
          HeritageButton(label: 'Truy xuất', onPressed: _submit),
          const SizedBox(height: 14),
          const Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              _Badge('Chuẩn OCOP 4 sao'),
              _Badge('Mã QR từng lô'),
              _Badge('Chỉ dẫn địa lý'),
              _Badge('Nguồn gốc rõ ràng'),
            ],
          ),
        ],
      ),
    ),
  );
}

class _Badge extends StatelessWidget {
  const _Badge(this.text);
  final String text;
  @override
  Widget build(BuildContext context) => DecoratedBox(
    decoration: BoxDecoration(
      border: Border.all(color: HeritageColors.mat),
      borderRadius: BorderRadius.circular(2),
    ),
    child: Padding(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
      child: Text(
        text,
        style: const TextStyle(color: HeritageColors.matSang, fontSize: 13),
      ),
    ),
  );
}

class _AccountCta extends StatelessWidget {
  const _AccountCta();
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.fromLTRB(16, 20, 16, 0),
    child: HeritageFrame(
      level: HeritageFrameLevel.functional,
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'ĐỒNG HÀNH CÙNG KHÁCH HÀNG',
            style: TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.2,
              color: AppColors.brandBrown,
            ),
          ),
          const SizedBox(height: 6),
          Text(
            'Một tài khoản cho trọn hành trình',
            style: Theme.of(context).textTheme.headlineMedium,
          ),
          const SizedBox(height: 8),
          const Text(
            'Quản lý đơn hàng, thông tin cá nhân và yêu cầu đổi trả ngay tại Gói Huế.',
          ),
          const SizedBox(height: 16),
          HeritageButton(
            label: 'Mở tài khoản của tôi',
            variant: HeritageButtonVariant.brown,
            onPressed: () => context.go(AppRoutes.account),
          ),
        ],
      ),
    ),
  );
}

/// Promotions are no longer a tab: Home links to them (and so does Account).
class _PromotionsBlock extends StatelessWidget {
  const _PromotionsBlock();
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 16),
    child: Semantics(
      button: true,
      label: 'Xem ưu đãi',
      child: InkWell(
        key: const Key('home-promotions'),
        onTap: () => context.push(AppRoutes.promotions),
        child: HeritageFrame(
          level: HeritageFrameLevel.functional,
          padding: const EdgeInsets.fromLTRB(16, 16, 32, 16),
          child: ExcludeSemantics(
            child: Row(
              children: [
                const HeritageIcon(
                  HeritageIconKind.gift,
                  size: 32,
                  color: AppColors.brickRed,
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Ưu đãi Tri Kỷ',
                        style: Theme.of(context).textTheme.titleLarge,
                      ),
                      const SizedBox(height: 2),
                      const Text(
                        'Mã đang phát hành và điều kiện áp dụng',
                        style: TextStyle(color: AppColors.textMuted),
                      ),
                    ],
                  ),
                ),
                const Text(
                  '→',
                  style: TextStyle(fontSize: 22, color: AppColors.brandBrown),
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );
}
