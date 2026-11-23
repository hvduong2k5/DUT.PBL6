import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';

/// Kicker (small caps with a diamond on both sides) + serif title + divider.
class HeritageSectionHeader extends StatelessWidget {
  const HeritageSectionHeader({
    super.key,
    required this.title,
    this.kicker,
    this.onDark = false,
  });

  final String title;
  final String? kicker;
  final bool onDark;

  @override
  Widget build(BuildContext context) {
    final fg = onDark ? HeritageColors.kem : HeritageColors.nauDam;
    final kickerColor = onDark ? HeritageColors.matSang : HeritageColors.nau;
    return Column(
      children: [
        if (kicker != null)
          Text(
            '❖  ${kicker!.toUpperCase()}  ❖',
            textAlign: TextAlign.center,
            style: TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w700,
              letterSpacing: 1.6,
              color: kickerColor,
            ),
          ),
        const SizedBox(height: 6),
        Text(
          title,
          textAlign: TextAlign.center,
          style: Theme.of(
            context,
          ).textTheme.headlineMedium?.copyWith(color: fg),
        ),
        const SizedBox(height: 10),
        Row(
          children: [
            Expanded(child: _rule(false)),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 10),
              child: SvgPicture.asset(
                HeritageAssets.divider,
                width: 28,
                height: 18,
                colorFilter: onDark
                    ? const ColorFilter.mode(
                        HeritageColors.matSang,
                        BlendMode.srcIn,
                      )
                    : null,
              ),
            ),
            Expanded(child: _rule(true)),
          ],
        ),
      ],
    );
  }

  Widget _rule(bool reverse) => Container(
    height: 1,
    decoration: BoxDecoration(
      gradient: LinearGradient(
        colors: [
          if (reverse) HeritageColors.nau else Colors.transparent,
          if (reverse) Colors.transparent else HeritageColors.nau,
        ],
      ),
    ),
  );
}
