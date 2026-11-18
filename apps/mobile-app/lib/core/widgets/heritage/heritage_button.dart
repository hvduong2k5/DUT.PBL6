import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';

enum HeritageButtonVariant { gold, brown, paper, ghost }

/// Kit button: >=48dp tall, 2dp radius. Default is honey gold.
class HeritageButton extends StatelessWidget {
  const HeritageButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.variant = HeritageButtonVariant.gold,
    this.loading = false,
    this.icon,
    this.expand = true,
  });

  final String label;
  final VoidCallback? onPressed;
  final HeritageButtonVariant variant;
  final bool loading;
  final IconData? icon;
  final bool expand;

  @override
  Widget build(BuildContext context) {
    final enabled = onPressed != null && !loading;
    final (bg, fg, side) = switch (variant) {
      HeritageButtonVariant.gold => (
        HeritageColors.mat,
        HeritageColors.nauCungDinh,
        HeritageColors.mat,
      ),
      HeritageButtonVariant.brown => (
        HeritageColors.nauDam,
        HeritageColors.kem,
        HeritageColors.mat,
      ),
      HeritageButtonVariant.paper => (
        HeritageColors.kem,
        HeritageColors.nauDam,
        HeritageColors.nau,
      ),
      HeritageButtonVariant.ghost => (
        Colors.transparent,
        HeritageColors.kem,
        HeritageColors.kem,
      ),
    };
    final style = ButtonStyle(
      minimumSize: const WidgetStatePropertyAll(Size(48, 48)),
      padding: const WidgetStatePropertyAll(
        EdgeInsets.symmetric(horizontal: 20),
      ),
      elevation: const WidgetStatePropertyAll(0),
      backgroundColor: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.disabled) ? HeritageColors.disabled : bg,
      ),
      foregroundColor: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.disabled) ? HeritageColors.phu : fg,
      ),
      side: WidgetStateProperty.resolveWith(
        (s) => BorderSide(
          color: s.contains(WidgetState.disabled)
              ? HeritageColors.disabled
              : side,
        ),
      ),
      shape: const WidgetStatePropertyAll(
        RoundedRectangleBorder(
          borderRadius: BorderRadius.all(Radius.circular(2)),
        ),
      ),
      textStyle: const WidgetStatePropertyAll(
        TextStyle(fontSize: 15, fontWeight: FontWeight.w700),
      ),
    );
    final content = Row(
      mainAxisSize: MainAxisSize.min,
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        if (loading)
          const SizedBox.square(
            dimension: 18,
            child: CircularProgressIndicator(strokeWidth: 2),
          )
        else if (icon != null)
          Icon(icon, size: 18),
        if (loading || icon != null) const SizedBox(width: 8),
        Flexible(child: Text(loading ? 'Đang xử lý...' : label)),
      ],
    );
    final button = FilledButton(
      onPressed: enabled ? onPressed : null,
      style: style,
      child: content,
    );
    return expand ? SizedBox(width: double.infinity, child: button) : button;
  }
}
