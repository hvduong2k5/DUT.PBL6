import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';

/// The kit logo used as the only decorative mark (kit rule: no custom icons).
class HeritageEmblem extends StatelessWidget {
  const HeritageEmblem({super.key, this.size = 56});
  final double size;

  @override
  Widget build(BuildContext context) => ExcludeSemantics(
    child: SvgPicture.asset(
      HeritageAssets.logo,
      width: size,
      height: size,
      fit: BoxFit.contain,
    ),
  );
}
