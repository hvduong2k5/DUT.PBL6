import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_app/app/router/routes.dart';

void safeBackOrHome(BuildContext context) {
  final router = GoRouter.of(context);
  if (router.canPop()) {
    router.pop();
  } else {
    router.go(AppRoutes.home);
  }
}

class SafeBackScope extends StatelessWidget {
  const SafeBackScope({super.key, required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    final canPop = GoRouter.of(context).canPop();
    return PopScope(
      canPop: canPop,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) GoRouter.of(context).go(AppRoutes.home);
      },
      child: child,
    );
  }
}
