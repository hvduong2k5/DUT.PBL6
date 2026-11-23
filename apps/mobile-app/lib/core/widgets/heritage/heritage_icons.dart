import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

/// The only line icons the kit allows in the shell (kit prototype
/// `home-v02/index.html`): search, cart, home, grid, gift.
enum HeritageIconKind { search, cart, home, grid, gift }

class HeritageIcon extends StatelessWidget {
  const HeritageIcon(this.kind, {super.key, this.size = 22, this.color});
  final HeritageIconKind kind;
  final double size;
  final Color? color;

  static const _paths = {
    HeritageIconKind.search:
        '<circle cx="11" cy="11" r="7"/><path d="M16 16l4.5 4.5" stroke-linecap="round"/>',
    HeritageIconKind.cart:
        '<path d="M4 8h16l-1.2 12H5.2L4 8z"/><path d="M8 8V6a4 4 0 0 1 8 0v2"/>',
    HeritageIconKind.home:
        '<path d="M3 10.5L12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1v-9.5z"/>',
    HeritageIconKind.grid:
        '<rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/>',
    HeritageIconKind.gift:
        '<rect x="3" y="8" width="18" height="13" rx="1"/><path d="M12 8v13M3 12h18M12 8c-2-3-5-3-5-1s3 1 5 1 5 1 5-1-3-2-5 1z"/>',
  };

  @override
  Widget build(BuildContext context) {
    final c = color ?? IconTheme.of(context).color ?? Colors.black;
    return ExcludeSemantics(
      child: SvgPicture.string(
        '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">${_paths[kind]}</svg>',
        width: size,
        height: size,
        colorMapper: _Mapper(c),
      ),
    );
  }
}

class _Mapper extends ColorMapper {
  const _Mapper(this.color);
  final Color color;
  @override
  Color substitute(
    String? id,
    String elementName,
    String attributeName,
    Color color_,
  ) => color;
}
