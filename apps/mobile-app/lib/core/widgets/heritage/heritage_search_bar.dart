import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:mobile_app/app/router/routes.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/header_actions.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_emblem.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_icons.dart';

/// Read-only search field: tapping it opens the dedicated search screen
/// (it never filters the current page).
class HeritageSearchBar extends StatelessWidget {
  const HeritageSearchBar({
    super.key,
    this.hint = 'Tìm mè xửng, trà sen…',
    this.query = '',
  });
  final String hint;
  final String query;

  @override
  Widget build(BuildContext context) => Semantics(
    button: true,
    label: 'Tìm kiếm sản phẩm',
    child: InkWell(
      key: const Key('search-bar'),
      onTap: () => context.push(
        query.isEmpty
            ? AppRoutes.search
            : '${AppRoutes.search}?q=${Uri.encodeQueryComponent(query)}',
      ),
      child: Container(
        height: 44,
        padding: const EdgeInsets.symmetric(horizontal: 12),
        decoration: BoxDecoration(
          color: HeritageColors.kem,
          borderRadius: BorderRadius.circular(2),
          border: Border.all(color: HeritageColors.mat),
        ),
        child: ExcludeSemantics(
          child: Row(
            children: [
              const HeritageIcon(
                HeritageIconKind.search,
                size: 20,
                color: HeritageColors.nau,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  query.isEmpty ? hint : query,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  // AppBar titles default to Playfair; the field uses body text.
                  style: GoogleFonts.beVietnamPro(
                    fontSize: 14,
                    color: query.isEmpty
                        ? HeritageColors.phu
                        : HeritageColors.muc,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}

/// Imperial app bar for tab roots: kit emblem + search bar (Shopee/Tiki
/// pattern). No back button and no side menu.
class HeritageSearchHeader extends StatelessWidget
    implements PreferredSizeWidget {
  const HeritageSearchHeader({super.key, this.query = ''});
  final String query;

  @override
  Size get preferredSize => const Size.fromHeight(64);

  @override
  Widget build(BuildContext context) => AppBar(
    automaticallyImplyLeading: false,
    toolbarHeight: 64,
    titleSpacing: 12,
    title: Row(
      children: [
        Semantics(label: 'Gói Huế', child: const HeritageEmblem(size: 34)),
        const SizedBox(width: 10),
        Expanded(child: HeritageSearchBar(query: query)),
      ],
    ),
    actions: const [
      HeaderCartAction(),
      HeaderNotificationsAction(),
      SizedBox(width: 4),
    ],
  );
}
