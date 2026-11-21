import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_frame.dart';

/// Money summary in a kit functional panel: labels left, amounts right; the
/// emphasised row (usually the total) uses Playfair in the price colour.
class HeritageAmountPanel extends StatelessWidget {
  const HeritageAmountPanel({super.key, required this.rows, this.total});
  final List<(String, String)> rows;
  final (String, String)? total;

  @override
  Widget build(BuildContext context) => HeritageFrame(
    level: HeritageFrameLevel.functional,
    padding: const EdgeInsets.fromLTRB(16, 12, 28, 12),
    child: Column(
      children: [
        for (final (label, value) in rows) _row(context, label, value, false),
        if (total != null) ...[
          const Divider(height: 16),
          _row(context, total!.$1, total!.$2, true),
        ],
      ],
    ),
  );

  Widget _row(BuildContext context, String label, String value, bool strong) =>
      Padding(
        padding: const EdgeInsets.symmetric(vertical: 6),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.baseline,
          textBaseline: TextBaseline.alphabetic,
          children: [
            Expanded(
              child: Text(
                label,
                style: strong
                    ? const TextStyle(fontWeight: FontWeight.w700)
                    : null,
              ),
            ),
            const SizedBox(width: 12),
            Text(
              value,
              textAlign: TextAlign.end,
              style: strong
                  ? GoogleFonts.playfairDisplay(
                      fontSize: 20,
                      fontWeight: FontWeight.w700,
                      color: HeritageColors.doChuSa,
                      fontFeatures: const [FontFeature.liningFigures()],
                    )
                  : Theme.of(context).textTheme.titleMedium,
            ),
          ],
        ),
      );
}
