import 'package:flutter/material.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_button.dart';

/// Secondary action: kit "paper" button (brown outline).
class AppSecondaryButton extends StatelessWidget {
  const AppSecondaryButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.icon,
  });
  final String label;
  final VoidCallback? onPressed;
  final IconData? icon;
  @override
  Widget build(BuildContext context) => HeritageButton(
    label: label,
    onPressed: onPressed,
    icon: icon,
    variant: HeritageButtonVariant.paper,
  );
}
