import 'package:flutter/material.dart';
import 'package:flutter_bloc/flutter_bloc.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_icons.dart';
import 'package:mobile_app/features/cart/presentation/cart_cubit.dart';

/// Cart shortcut for top bars; the badge follows the shared cart state.
class HeaderCartAction extends StatelessWidget {
  const HeaderCartAction({super.key});

  @override
  Widget build(BuildContext context) {
    final count = context.select((CartCubit c) => c.state.cart.totalItems);
    return SizedBox.square(
      dimension: 48,
      child: IconButton(
        key: const Key('header-cart'),
        tooltip: count > 0 ? 'Giỏ hàng, $count sản phẩm' : 'Giỏ hàng',
        onPressed: () => context.go(AppRoutes.cart),
        icon: Badge(
          isLabelVisible: count > 0,
          label: Text(count > 99 ? '99+' : '$count'),
          backgroundColor: HeritageColors.mat,
          textColor: HeritageColors.nauCungDinh,
          child: const HeritageIcon(
            HeritageIconKind.cart,
            color: HeritageColors.kem,
          ),
        ),
      ),
    );
  }
}

/// Notifications live in the top bar (not a bottom tab); opens the list.
class HeaderNotificationsAction extends StatelessWidget {
  const HeaderNotificationsAction({super.key});

  @override
  Widget build(BuildContext context) => SizedBox.square(
    dimension: 48,
    child: IconButton(
      key: const Key('header-notifications'),
      tooltip: 'Thông báo',
      onPressed: () => context.push(AppRoutes.notifications),
      icon: const Icon(Icons.notifications_none, color: HeritageColors.kem),
    ),
  );
}
