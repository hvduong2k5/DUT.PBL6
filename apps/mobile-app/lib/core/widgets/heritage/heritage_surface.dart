import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';

enum HeritageSurfaceKind { paper, parchment, imperial }

/// Page background: kit paper texture tiled at 600px under a tint overlay.
class HeritageSurface extends StatelessWidget {
  const HeritageSurface({
    super.key,
    required this.child,
    this.kind = HeritageSurfaceKind.paper,
  });

  final Widget child;
  final HeritageSurfaceKind kind;

  @override
  Widget build(BuildContext context) {
    switch (kind) {
      case HeritageSurfaceKind.imperial:
        return DecoratedBox(
          decoration: const BoxDecoration(
            gradient: RadialGradient(
              center: Alignment(0, -0.6),
              radius: 1.2,
              colors: [Color(0xFF4A2D1C), HeritageColors.nauCungDinh],
            ),
          ),
          child: child,
        );
      case HeritageSurfaceKind.paper:
      case HeritageSurfaceKind.parchment:
        final tint = kind == HeritageSurfaceKind.paper
            ? HeritageColors.kem.withValues(alpha: 0.9)
            : HeritageColors.kemAm.withValues(alpha: 0.8);
        return DecoratedBox(
          decoration: BoxDecoration(
            color: tint,
            image: const DecorationImage(
              image: AssetImage(HeritageAssets.paper),
              repeat: ImageRepeat.repeat,
              scale: 1.6,
            ),
          ),
          child: DecoratedBox(
            decoration: BoxDecoration(color: tint),
            child: child,
          ),
        );
    }
  }
}
