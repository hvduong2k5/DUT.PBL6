import 'package:flutter/material.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_frame.dart';

/// A step or block of a form: kit functional panel (1px brown border and a
/// folded corner) with a serif title.
class HeritageFormSection extends StatelessWidget {
  const HeritageFormSection({
    super.key,
    required this.title,
    required this.children,
  });
  final String title;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 12),
    child: HeritageFrame(
      level: HeritageFrameLevel.functional,
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            // Keep the title clear of the 22px folded corner.
            padding: const EdgeInsets.only(right: 20),
            child: Text(title, style: Theme.of(context).textTheme.titleLarge),
          ),
          const SizedBox(height: 16),
          ...children,
        ],
      ),
    ),
  );
}
