import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_button.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_frame.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_section_header.dart';
import 'package:mobile_app/core/widgets/heritage/heritage_surface.dart';

void main() {
  testWidgets('HeritageButton is at least 48dp tall and fires onPressed', (
    tester,
  ) async {
    var taps = 0;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: HeritageButton(label: 'Mua ngay', onPressed: () => taps++),
        ),
      ),
    );
    expect(
      tester.getSize(find.byType(FilledButton)).height,
      greaterThanOrEqualTo(48),
    );
    await tester.tap(find.text('Mua ngay'));
    expect(taps, 1);
  });

  testWidgets('disabled HeritageButton does not fire', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: HeritageButton(label: 'Mua', onPressed: null)),
      ),
    );
    await tester.tap(find.text('Mua'), warnIfMissed: false);
    expect(
      tester.widget<FilledButton>(find.byType(FilledButton)).onPressed,
      isNull,
    );
  });

  testWidgets('frames render at every level with Vietnamese text', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: Column(
              children: [
                for (final level in HeritageFrameLevel.values)
                  HeritageFrame(level: level, child: const Text('Cà phê Huế')),
                const HeritageSectionHeader(
                  kicker: 'Chuyện Huế',
                  title: 'Quà Huế',
                ),
                const SizedBox(
                  height: 40,
                  child: HeritageSurface(
                    kind: HeritageSurfaceKind.imperial,
                    child: SizedBox.expand(),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
    expect(tester.takeException(), isNull);
    expect(find.text('Cà phê Huế'), findsNWidgets(4));
  });

  test('step path stays inside the box', () {
    final b = heritageStepPath(const Size(200, 100), 1).getBounds();
    expect(b.left, 1);
    expect(b.right, 199);
    expect(HeritageColors.mat, const Color(0xFFD4A24C));
  });
}
