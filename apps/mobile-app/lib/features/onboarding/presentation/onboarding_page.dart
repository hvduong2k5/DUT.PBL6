import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/storage/onboarding_store.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_button.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_emblem.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_surface.dart';

class OnboardingPage extends StatefulWidget {
  const OnboardingPage({super.key, required this.store});

  final OnboardingStore store;

  @override
  State<OnboardingPage> createState() => _OnboardingPageState();
}

class _OnboardingPageState extends State<OnboardingPage> {
  final _controller = PageController();
  var _page = 0;
  var _finishing = false;

  static const _slides = [
    (
      image: 'assets/heritage/photos/hero-studio.jpg',
      title: 'Tinh hoa quà Huế',
      message: 'Khám phá mè xửng, trà và những món quà mang hương vị Cố Đô.',
    ),
    (
      image: 'assets/heritage/photos/hero-tea-table.jpg',
      title: 'Mua sắm thật thuận tiện',
      message:
          'Chọn sản phẩm, theo dõi đơn hàng và nhận thông báo trong một nơi.',
    ),
    (
      image: 'assets/heritage/photos/sp-hop-qua-doan-vien.jpg',
      title: 'Bắt đầu theo cách của bạn',
      message:
          'Đăng nhập để quản lý tài khoản hoặc tiếp tục mua sắm với tư cách khách.',
    ),
  ];

  Future<void> _finish(String route) async {
    if (_finishing) return;
    setState(() => _finishing = true);
    await widget.store.complete();
    if (mounted) context.go(route);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => PopScope(
    canPop: false,
    onPopInvokedWithResult: (didPop, _) {
      if (!didPop) _finish(AppRoutes.home);
    },
    child: Scaffold(
      backgroundColor: HeritageColors.nauCungDinh,
      body: HeritageSurface(
        kind: HeritageSurfaceKind.imperial,
        child: SafeArea(
          child: Column(
            children: [
              Expanded(
                child: PageView.builder(
                  controller: _controller,
                  itemCount: _slides.length,
                  onPageChanged: (value) => setState(() => _page = value),
                  itemBuilder: (context, index) => _OnboardingSlide(
                    image: _slides[index].image,
                    title: _slides[index].title,
                    message: _slides[index].message,
                  ),
                ),
              ),
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: List.generate(
                  _slides.length,
                  (index) => AnimatedContainer(
                    key: ValueKey('onboarding-dot-$index'),
                    duration: HeritageMotion.duration,
                    curve: HeritageMotion.curve,
                    width: index == _page ? 24 : 8,
                    height: 8,
                    margin: const EdgeInsets.symmetric(horizontal: 4),
                    decoration: BoxDecoration(
                      color: index == _page
                          ? HeritageColors.mat
                          : HeritageColors.kem.withValues(alpha: 0.4),
                      borderRadius: BorderRadius.circular(99),
                    ),
                  ),
                ),
              ),
              Padding(
                padding: const EdgeInsets.all(AppSpacing.lg),
                child: _page < _slides.length - 1
                    ? HeritageButton(
                        label: 'Tiếp tục',
                        icon: Icons.arrow_forward,
                        onPressed: () => _controller.nextPage(
                          duration: HeritageMotion.duration,
                          curve: HeritageMotion.curve,
                        ),
                      )
                    : Column(
                        children: [
                          HeritageButton(
                            label: 'Đăng nhập',
                            loading: _finishing,
                            onPressed: () => _finish(AppRoutes.login),
                          ),
                          const SizedBox(height: AppSpacing.sm),
                          HeritageButton(
                            label: 'Đăng ký',
                            variant: HeritageButtonVariant.ghost,
                            onPressed: _finishing
                                ? null
                                : () => _finish(AppRoutes.register),
                          ),
                          TextButton(
                            style: TextButton.styleFrom(
                              foregroundColor: HeritageColors.kem,
                            ),
                            onPressed: _finishing
                                ? null
                                : () => _finish(AppRoutes.home),
                            child: const Text('Tiếp tục với tư cách khách'),
                          ),
                        ],
                      ),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}

class _OnboardingSlide extends StatelessWidget {
  const _OnboardingSlide({
    required this.image,
    required this.title,
    required this.message,
  });

  final String image;
  final String title;
  final String message;

  @override
  Widget build(BuildContext context) => Stack(
    fit: StackFit.expand,
    children: [
      // Photo dissolves into the imperial background (no hard seam).
      ShaderMask(
        blendMode: BlendMode.dstIn,
        shaderCallback: (rect) => const LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [Color(0xFF000000), Color(0x99000000), Color(0x00000000)],
          stops: [0, 0.55, 1],
        ).createShader(rect),
        child: Image.asset(
          image,
          fit: BoxFit.cover,
          alignment: Alignment.topCenter,
        ),
      ),
      Padding(
        padding: const EdgeInsets.fromLTRB(
          AppSpacing.xl,
          AppSpacing.lg,
          AppSpacing.xl,
          AppSpacing.lg,
        ),
        child: Column(
          children: [
            const HeritageEmblem(size: 56),
            const Spacer(),
            Text(
              title,
              style: Theme.of(context).textTheme.headlineLarge?.copyWith(
                color: HeritageColors.kem,
                fontWeight: FontWeight.w700,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: AppSpacing.sm),
            Text(
              message,
              textAlign: TextAlign.center,
              style: const TextStyle(
                color: HeritageColors.kemDam,
                fontSize: 15,
                height: 1.5,
              ),
            ),
          ],
        ),
      ),
    ],
  );
}
