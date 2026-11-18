import 'dart:async';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_button.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_frame.dart';

class HeritageBannerSlide {
  const HeritageBannerSlide({
    required this.image,
    required this.title,
    this.kicker,
    this.subtitle,
    this.actionLabel,
    this.onAction,
  });
  final ImageProvider image;
  final String title;
  final String? kicker;
  final String? subtitle;
  final String? actionLabel;
  final VoidCallback? onAction;
}

/// Hero banner: imperial stage + double frame, photo fading into the brown
/// background (no hard seam), auto-advance every 3s, dots bottom-centre.
class HeritageBanner extends StatefulWidget {
  const HeritageBanner({
    super.key,
    required this.slides,
    this.height = 300,
    this.interval = const Duration(seconds: 3),
  });
  final List<HeritageBannerSlide> slides;
  final double height;
  final Duration interval;

  @override
  State<HeritageBanner> createState() => _HeritageBannerState();
}

class _HeritageBannerState extends State<HeritageBanner> {
  final _controller = PageController();
  Timer? _timer;
  int _index = 0;
  bool _reduceMotion = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _reduceMotion = MediaQuery.disableAnimationsOf(context);
    _restart();
  }

  void _restart() {
    _timer?.cancel();
    if (widget.slides.length < 2 || _reduceMotion) return;
    _timer = Timer.periodic(widget.interval, (_) => _go(_index + 1));
  }

  void _go(int target) {
    if (!_controller.hasClients) return;
    final next = target % widget.slides.length;
    final distance = (next - _index).abs();
    if (_reduceMotion || distance > 1) {
      _controller.jumpToPage(next);
    } else {
      _controller.animateToPage(
        next,
        duration: HeritageMotion.duration * 2,
        curve: HeritageMotion.curve,
      );
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (widget.slides.isEmpty) return const SizedBox.shrink();
    return SizedBox(
      height: widget.height,
      child: DecoratedBox(
        decoration: const BoxDecoration(color: HeritageColors.nauCungDinh),
        child: Stack(
          fit: StackFit.expand,
          children: [
            Listener(
              onPointerDown: (_) => _timer?.cancel(),
              onPointerUp: (_) => _restart(),
              child: PageView.builder(
                controller: _controller,
                itemCount: widget.slides.length,
                onPageChanged: (i) => setState(() => _index = i),
                itemBuilder: (_, i) => _Slide(slide: widget.slides[i]),
              ),
            ),
            const IgnorePointer(
              child: Padding(
                padding: EdgeInsets.all(8),
                child: CustomPaint(
                  painter: HeritageFramePainter(HeritageFrameLevel.imperial),
                ),
              ),
            ),
            if (widget.slides.length > 1) ...[
              Align(
                alignment: Alignment.centerLeft,
                child: _Arrow(
                  icon: Icons.chevron_left,
                  label: 'Slide trước',
                  onTap: () => _go(_index - 1 + widget.slides.length),
                ),
              ),
              Align(
                alignment: Alignment.centerRight,
                child: _Arrow(
                  icon: Icons.chevron_right,
                  label: 'Slide sau',
                  onTap: () => _go(_index + 1),
                ),
              ),
              Align(
                alignment: Alignment.bottomCenter,
                child: Padding(
                  padding: const EdgeInsets.only(bottom: 14),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      for (var i = 0; i < widget.slides.length; i++)
                        AnimatedContainer(
                          duration: HeritageMotion.duration,
                          curve: HeritageMotion.curve,
                          margin: const EdgeInsets.symmetric(horizontal: 3),
                          width: i == _index ? 14 : 6,
                          height: 6,
                          decoration: BoxDecoration(
                            color: i == _index
                                ? HeritageColors.mat
                                : HeritageColors.kem.withValues(alpha: 0.5),
                            borderRadius: BorderRadius.circular(3),
                          ),
                        ),
                    ],
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _Slide extends StatelessWidget {
  const _Slide({required this.slide});
  final HeritageBannerSlide slide;

  @override
  Widget build(BuildContext context) => Stack(
    fit: StackFit.expand,
    children: [
      // Photo fades into the stage colour: top and left edges dissolve.
      ShaderMask(
        blendMode: BlendMode.dstIn,
        shaderCallback: (rect) => const LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [Color(0x00000000), Color(0x99000000), Color(0xFF000000)],
          stops: [0, 0.45, 1],
        ).createShader(rect),
        child: Opacity(
          opacity: 0.55,
          child: Image(
            image: slide.image,
            fit: BoxFit.cover,
            errorBuilder: (_, _, _) => const SizedBox.shrink(),
          ),
        ),
      ),
      Padding(
        padding: const EdgeInsets.fromLTRB(40, 28, 40, 36),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.end,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (slide.kicker != null)
              Text(
                '❖  ${slide.kicker!.toUpperCase()}',
                style: const TextStyle(
                  color: HeritageColors.matSang,
                  fontSize: 13,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 1.6,
                ),
              ),
            const SizedBox(height: 8),
            Text(
              slide.title,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                color: HeritageColors.kem,
                fontWeight: FontWeight.w700,
              ),
            ),
            if (slide.subtitle != null) ...[
              const SizedBox(height: 6),
              Text(
                slide.subtitle!,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  color: HeritageColors.kemDam,
                  fontSize: 14,
                  height: 1.5,
                ),
              ),
            ],
            if (slide.actionLabel != null) ...[
              const SizedBox(height: 14),
              HeritageButton(
                label: slide.actionLabel!,
                onPressed: slide.onAction,
                expand: false,
              ),
            ],
          ],
        ),
      ),
    ],
  );
}

/// Round blurred chip; faint because touch devices have no hover state.
class _Arrow extends StatelessWidget {
  const _Arrow({required this.icon, required this.label, required this.onTap});
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: 12),
    child: Semantics(
      button: true,
      label: label,
      child: ClipOval(
        child: BackdropFilter(
          filter: ui.ImageFilter.blur(sigmaX: 6, sigmaY: 6),
          child: Material(
            color: HeritageColors.nauCungDinh.withValues(alpha: 0.35),
            child: InkWell(
              onTap: onTap,
              child: SizedBox.square(
                dimension: 44,
                child: Icon(
                  icon,
                  color: HeritageColors.kem.withValues(alpha: 0.75),
                ),
              ),
            ),
          ),
        ),
      ),
    ),
  );
}
