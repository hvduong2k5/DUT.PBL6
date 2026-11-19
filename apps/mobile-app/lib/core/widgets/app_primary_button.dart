import 'package:flutter/material.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_button.dart';

/// Primary action: kit "brown" button (cream text, gold frame).
class AppPrimaryButton extends StatelessWidget {
  const AppPrimaryButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.loading = false,
    this.icon,
  });
  final String label;
  final VoidCallback? onPressed;
  final bool loading;
  final IconData? icon;

  @override
  Widget build(BuildContext context) => HeritageButton(
    label: label,
    onPressed: onPressed,
    loading: loading,
    icon: icon,
    variant: HeritageButtonVariant.brown,
  );
}
