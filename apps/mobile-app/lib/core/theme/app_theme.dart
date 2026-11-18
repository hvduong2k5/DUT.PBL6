import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_radius.dart';
import 'package:mobile_app/core/theme/heritage_tokens.dart';

/// Goi Hue heritage theme: Playfair Display headings, Be Vietnam Pro body.
abstract final class AppTheme {
  static const _rect = RoundedRectangleBorder(
    borderRadius: BorderRadius.all(Radius.circular(AppRadius.small)),
  );

  static TextStyle _heading(double size, FontWeight weight, {double? height}) =>
      GoogleFonts.playfairDisplay(
        fontSize: size,
        height: height,
        fontWeight: weight,
        color: AppColors.headingBrown,
        fontFeatures: const [FontFeature.liningFigures()],
      );

  static OutlineInputBorder _border(Color color, [double width = 1]) =>
      OutlineInputBorder(
        borderRadius: BorderRadius.circular(AppRadius.small),
        borderSide: BorderSide(color: color, width: width),
      );

  static ThemeData get light {
    final scheme = ColorScheme.fromSeed(
      seedColor: AppColors.brandBrown,
      brightness: Brightness.light,
      primary: AppColors.headingBrown,
      onPrimary: HeritageColors.kem,
      secondary: AppColors.honeyGold,
      onSecondary: HeritageColors.nauCungDinh,
      surface: AppColors.ivory,
      onSurface: AppColors.text,
      error: HeritageColors.loi,
      outline: AppColors.brandBrown,
      outlineVariant: AppColors.border,
    );
    final body = GoogleFonts.beVietnamProTextTheme();
    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: AppColors.ivory,
      fontFamily: GoogleFonts.beVietnamPro().fontFamily,
      textTheme: body.copyWith(
        headlineLarge: _heading(26, FontWeight.w700, height: 34 / 26),
        headlineMedium: _heading(22, FontWeight.w600, height: 30 / 22),
        titleLarge: _heading(20, FontWeight.w600),
        titleMedium: body.titleMedium?.copyWith(
          fontSize: 16,
          fontWeight: FontWeight.w600,
          color: AppColors.headingBrown,
        ),
        bodyLarge: body.bodyLarge?.copyWith(
          fontSize: 16,
          height: 1.5,
          color: AppColors.text,
        ),
        bodyMedium: body.bodyMedium?.copyWith(
          fontSize: 14,
          height: 1.5,
          color: AppColors.text,
        ),
        bodySmall: body.bodySmall?.copyWith(
          fontSize: 13,
          height: 1.5,
          color: AppColors.textMuted,
        ),
        labelLarge: body.labelLarge?.copyWith(
          fontSize: 13,
          fontWeight: FontWeight.w700,
        ),
      ),
      appBarTheme: AppBarTheme(
        backgroundColor: HeritageColors.nauCungDinh,
        foregroundColor: HeritageColors.kem,
        elevation: 0,
        scrolledUnderElevation: 0,
        surfaceTintColor: Colors.transparent,
        iconTheme: const IconThemeData(color: HeritageColors.kem),
        titleTextStyle: GoogleFonts.playfairDisplay(
          fontSize: 20,
          fontWeight: FontWeight.w600,
          color: HeritageColors.kem,
        ),
        shape: const Border(
          bottom: BorderSide(color: HeritageColors.mat, width: 3),
        ),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          minimumSize: const Size(48, 48),
          backgroundColor: AppColors.headingBrown,
          foregroundColor: HeritageColors.kem,
          disabledBackgroundColor: HeritageColors.disabled,
          side: const BorderSide(color: HeritageColors.mat),
          shape: _rect,
          textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w700),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(48, 48),
          foregroundColor: AppColors.headingBrown,
          side: const BorderSide(color: AppColors.brandBrown),
          shape: _rect,
          textStyle: const TextStyle(fontSize: 15, fontWeight: FontWeight.w700),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          minimumSize: const Size(48, 48),
          foregroundColor: AppColors.headingBrown,
          shape: _rect,
        ),
      ),
      chipTheme: ChipThemeData(
        selectedColor: AppColors.headingBrown,
        backgroundColor: AppColors.surface,
        side: const BorderSide(color: AppColors.border),
        labelStyle: GoogleFonts.beVietnamPro(
          fontSize: 13,
          fontWeight: FontWeight.w500,
          color: AppColors.text,
        ),
        secondaryLabelStyle: GoogleFonts.beVietnamPro(
          fontSize: 13,
          fontWeight: FontWeight.w600,
          color: HeritageColors.kem,
        ),
        shape: _rect,
        showCheckmark: false,
      ),
      dividerTheme: const DividerThemeData(
        color: AppColors.border,
        thickness: 1,
        space: 0,
      ),
      cardTheme: CardThemeData(
        color: AppColors.surface,
        elevation: 0,
        shadowColor: const Color(0x14352014),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppRadius.large),
          side: const BorderSide(color: AppColors.border),
        ),
      ),
      bottomSheetTheme: const BottomSheetThemeData(
        backgroundColor: AppColors.ivory,
        surfaceTintColor: Colors.transparent,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: HeritageColors.kem,
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 14,
        ),
        floatingLabelBehavior: FloatingLabelBehavior.always,
        border: _border(AppColors.brandBrown),
        enabledBorder: _border(AppColors.brandBrown),
        focusedBorder: _border(AppColors.headingBrown, 2),
        errorBorder: _border(HeritageColors.loi),
        focusedErrorBorder: _border(HeritageColors.loi, 2),
        labelStyle: const TextStyle(color: AppColors.text, fontSize: 14),
        hintStyle: const TextStyle(color: AppColors.textMuted),
        errorStyle: const TextStyle(color: HeritageColors.loi, fontSize: 13),
      ),
    );
  }
}
