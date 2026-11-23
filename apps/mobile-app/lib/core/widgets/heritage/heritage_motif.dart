import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';

/// Decorative kit motif, placed absolutely at a page edge, below content.
/// Use inside a [Stack]. Never receives pointer events.
class HeritageMotif extends StatelessWidget {
  const HeritageMotif(
    this.name, {
    super.key,
    this.width = 160,
    this.opacity = 0.4,
    this.alignment = Alignment.topRight,
  });

  /// File name without extension, e.g. `03-canh-tra` or `01-hien-nha-phai`.
  final String name;
  final double width;
  final double opacity;
  final AlignmentGeometry alignment;

  @override
  Widget build(BuildContext context) => Positioned.fill(
    child: IgnorePointer(
      child: ExcludeSemantics(
        child: Align(
          alignment: alignment,
          child: Opacity(
            opacity: opacity,
            child: Image.asset(
              HeritageAssets.motif(name),
              width: width,
              fit: BoxFit.contain,
              errorBuilder: (_, _, _) => const SizedBox.shrink(),
            ),
          ),
        ),
      ),
    ),
  );
}
