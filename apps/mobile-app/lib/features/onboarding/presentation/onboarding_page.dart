import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/storage/onboarding_store.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_spacing.dart';
import 'package:mobile_app/core/widgets/app_primary_button.dart';
import 'package:mobile_app/core/widgets/app_secondary_button.dart';

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
      icon: Icons.storefront_outlined,
      title: 'Tinh hoa quà Huế',
      message: 'Khám phá mè xửng, trà và những món quà mang hương vị Cố Đô.',
    ),
    (
      icon: Icons.local_shipping_outlined,
      title: 'Mua sắm thật thuận tiện',
      message:
          'Chọn sản phẩm, theo dõi đơn hàng và nhận thông báo trong một nơi.',
    ),
    (
      icon: Icons.person_outline,
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
      body: SafeArea(
        child: Column(
          children: [
            Expanded(
              child: PageView.builder(
                controller: _controller,
                itemCount: _slides.length,
                onPageChanged: (value) => setState(() => _page = value),
                itemBuilder: (context, index) => _OnboardingSlide(
                  icon: _slides[index].icon,
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
                  duration: const Duration(milliseconds: 180),
                  width: index == _page ? 24 : 8,
                  height: 8,
                  margin: const EdgeInsets.symmetric(horizontal: 4),
                  decoration: BoxDecoration(
                    color: index == _page
                        ? AppColors.brandBrown
                        : AppColors.border,
                    borderRadius: BorderRadius.circular(99),
                  ),
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.all(AppSpacing.lg),
              child: _page < _slides.length - 1
                  ? SizedBox(
                      width: double.infinity,
                      child: AppPrimaryButton(
                        label: 'Tiếp tục',
                        icon: Icons.arrow_forward,
                        onPressed: () => _controller.nextPage(
                          duration: const Duration(milliseconds: 260),
                          curve: Curves.easeOut,
                        ),
                      ),
                    )
                  : Column(
                      children: [
                        SizedBox(
                          width: double.infinity,
                          child: AppPrimaryButton(
                            label: 'Đăng nhập',
                            loading: _finishing,
                            onPressed: () => _finish(AppRoutes.login),
                          ),
                        ),
                        const SizedBox(height: AppSpacing.sm),
                        SizedBox(
                          width: double.infinity,
                          child: AppSecondaryButton(
                            label: 'Đăng ký',
                            onPressed: _finishing
                                ? null
                                : () => _finish(AppRoutes.register),
                          ),
                        ),
                        TextButton(
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
  );
}

class _OnboardingSlide extends StatelessWidget {
  const _OnboardingSlide({
    required this.icon,
    required this.title,
    required this.message,
  });

  final IconData icon;
  final String title;
  final String message;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: AppSpacing.xl),
    child: Column(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        Container(
          width: 132,
          height: 132,
          decoration: BoxDecoration(
            color: AppColors.cream,
            shape: BoxShape.circle,
            border: Border.all(color: AppColors.honeyGold),
          ),
          child: Icon(icon, size: 54, color: AppColors.brandBrown),
        ),
        const SizedBox(height: AppSpacing.xl),
        Text(
          title,
          style: Theme.of(context).textTheme.headlineLarge,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: AppSpacing.sm),
        Text(message, textAlign: TextAlign.center),
      ],
    ),
  );
}
