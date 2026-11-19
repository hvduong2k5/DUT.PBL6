import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';

/// Frame levels of the kit. Geometry mirrors the kit SVGs (48x48, stepped
/// corners with offsets 1/5/9) so frames scale to any size without distortion.
enum HeritageFrameLevel { imperial, plaque, card, functional }

/// Stepped-corner polygon used by every frame, inset by [b] logical pixels.
Path heritageStepPath(Size size, double b) {
  final w = size.width;
  final h = size.height;
  final a = b + 4;
  final c = b + 8;
  return Path()
    ..moveTo(b, c)
    ..lineTo(a, c)
    ..lineTo(a, a)
    ..lineTo(c, a)
    ..lineTo(c, b)
    ..lineTo(w - c, b)
    ..lineTo(w - c, a)
    ..lineTo(w - a, a)
    ..lineTo(w - a, c)
    ..lineTo(w - b, c)
    ..lineTo(w - b, h - c)
    ..lineTo(w - a, h - c)
    ..lineTo(w - a, h - a)
    ..lineTo(w - c, h - a)
    ..lineTo(w - c, h - b)
    ..lineTo(c, h - b)
    ..lineTo(c, h - a)
    ..lineTo(a, h - a)
    ..lineTo(a, h - c)
    ..lineTo(b, h - c)
    ..close();
}

class HeritageFramePainter extends CustomPainter {
  const HeritageFramePainter(this.level, {this.hovered = false});
  final HeritageFrameLevel level;

  /// For [HeritageFrameLevel.card]: switches to the plaque (double) frame.
  final bool hovered;

  Paint _stroke(Color color, double width) => Paint()
    ..style = PaintingStyle.stroke
    ..strokeWidth = width
    ..strokeJoin = StrokeJoin.miter
    ..color = color;

  @override
  void paint(Canvas canvas, Size size) {
    switch (level) {
      case HeritageFrameLevel.imperial:
        canvas.drawPath(
          heritageStepPath(size, 1),
          _stroke(HeritageColors.kem, 1),
        );
        canvas.drawPath(
          heritageStepPath(size, 4),
          _stroke(HeritageColors.kem, 1),
        );
      case HeritageFrameLevel.plaque:
        _plaque(canvas, size);
      case HeritageFrameLevel.card:
        if (hovered) {
          _plaque(canvas, size);
        } else {
          canvas.drawPath(
            heritageStepPath(size, 1),
            _stroke(HeritageColors.nau, 2.2),
          );
        }
      case HeritageFrameLevel.functional:
        final rect = (Offset.zero & size).deflate(0.5);
        canvas.drawRect(rect, _stroke(HeritageColors.nau, 1));
        const fold = 22.0;
        final x = size.width - 0.5;
        canvas.drawPath(
          Path()
            ..moveTo(x - fold, 0.5)
            ..lineTo(x, 0.5)
            ..lineTo(x, fold)
            ..close(),
          Paint()..color = HeritageColors.mat,
        );
        canvas.drawPath(
          Path()
            ..moveTo(x - fold, 0.5)
            ..lineTo(x - fold, fold)
            ..lineTo(x, fold)
            ..close(),
          Paint()..color = HeritageColors.kemDam,
        );
        canvas.drawLine(
          Offset(x - fold, 0.5),
          Offset(x, fold),
          _stroke(HeritageColors.nau, 1),
        );
    }
  }

  void _plaque(Canvas canvas, Size size) {
    canvas.drawPath(
      heritageStepPath(size, 1),
      _stroke(HeritageColors.nau, 2.2),
    );
    canvas.drawPath(
      heritageStepPath(size, 4.5),
      _stroke(HeritageColors.mat, 1),
    );
  }

  @override
  bool shouldRepaint(HeritageFramePainter old) =>
      old.level != level || old.hovered != hovered;
}

/// Clips children to the stepped-corner outline of a frame.
class HeritageStepClipper extends CustomClipper<Path> {
  const HeritageStepClipper();
  @override
  Path getClip(Size size) => heritageStepPath(size, 1);
  @override
  bool shouldReclip(HeritageStepClipper old) => false;
}

/// A surface with a kit frame drawn over it.
class HeritageFrame extends StatelessWidget {
  const HeritageFrame({
    super.key,
    required this.child,
    this.level = HeritageFrameLevel.plaque,
    this.padding = const EdgeInsets.all(16),
    this.hovered = false,
    this.color,
  });

  final Widget child;
  final HeritageFrameLevel level;
  final EdgeInsetsGeometry padding;
  final bool hovered;
  final Color? color;

  Color get _fill =>
      color ??
      switch (level) {
        HeritageFrameLevel.imperial => HeritageColors.nauCungDinh,
        HeritageFrameLevel.functional => HeritageColors.kem,
        _ => HeritageColors.kem,
      };

  @override
  Widget build(BuildContext context) {
    final stepped = level != HeritageFrameLevel.functional;
    Widget body = DecoratedBox(
      decoration: BoxDecoration(color: _fill),
      child: Material(
        type: MaterialType.transparency,
        child: Padding(padding: padding, child: child),
      ),
    );
    if (stepped) {
      body = ClipPath(clipper: const HeritageStepClipper(), child: body);
    }
    return CustomPaint(
      foregroundPainter: HeritageFramePainter(level, hovered: hovered),
      child: body,
    );
  }
}
