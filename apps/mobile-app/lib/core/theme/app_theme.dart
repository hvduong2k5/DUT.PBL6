import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:mobile_app/core/theme/app_colors.dart';
import 'package:mobile_app/core/theme/app_radius.dart';

abstract final class AppTheme {
  static ThemeData get light {
    final scheme = ColorScheme.fromSeed(
      seedColor: AppColors.brandBrown,
      brightness: Brightness.light,
      primary: AppColors.brandBrown,
      secondary: AppColors.honeyGold,
      surface: AppColors.ivory,
      error: AppColors.brickRed,
    );
    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: AppColors.ivory,
      fontFamily: GoogleFonts.beVietnamPro().fontFamily,
      textTheme: TextTheme(
        headlineLarge: TextStyle(
          fontFamily: GoogleFonts.notoSerif().fontFamily,
          fontSize: 26,
          height: 34 / 26,
          fontWeight: FontWeight.w600,
          color: AppColors.headingBrown,
        ),
        headlineMedium: TextStyle(
          fontFamily: GoogleFonts.notoSerif().fontFamily,
          fontSize: 22,
          fontWeight: FontWeight.w600,
          color: AppColors.headingBrown,
        ),
        titleLarge: TextStyle(
          fontFamily: GoogleFonts.notoSerif().fontFamily,
          fontSize: 20,
          fontWeight: FontWeight.w600,
          color: AppColors.headingBrown,
        ),
        titleMedium: TextStyle(
          fontSize: 16,
          fontWeight: FontWeight.w600,
          color: AppColors.headingBrown,
        ),
        bodyLarge: TextStyle(fontSize: 16, height: 1.5, color: AppColors.text),
        bodyMedium: TextStyle(fontSize: 14, height: 1.5, color: AppColors.text),
        bodySmall: TextStyle(
          fontSize: 12,
          height: 1.5,
          color: AppColors.textMuted,
        ),
        labelLarge: TextStyle(fontSize: 13, fontWeight: FontWeight.w700),
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: AppColors.ivory,
        foregroundColor: AppColors.headingBrown,
        elevation: 0,
        surfaceTintColor: Colors.transparent,
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          backgroundColor: AppColors.honeyGold,
          foregroundColor: AppColors.ctaText,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppRadius.large),
          ),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: AppColors.brandBrown,
          side: const BorderSide(color: AppColors.brandBrown),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(AppRadius.large),
          ),
        ),
      ),
      chipTheme: ChipThemeData(
        selectedColor: AppColors.brandBrown,
        backgroundColor: AppColors.surface,
        side: const BorderSide(color: AppColors.border),
        labelStyle: const TextStyle(
          fontFamily: 'Be Vietnam Pro',
          fontSize: 13,
          fontWeight: FontWeight.w500,
          color: AppColors.text,
        ),
        secondaryLabelStyle: const TextStyle(
          fontFamily: 'Be Vietnam Pro',
          fontSize: 13,
          fontWeight: FontWeight.w600,
          color: Colors.white,
        ),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppRadius.card),
        ),
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
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(AppRadius.large),
          side: const BorderSide(color: AppColors.border),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: Colors.white,
        contentPadding: const EdgeInsets.symmetric(
          horizontal: 16,
          vertical: 14,
        ),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppRadius.large),
          borderSide: const BorderSide(color: AppColors.border),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppRadius.large),
          borderSide: const BorderSide(color: AppColors.border),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(AppRadius.large),
          borderSide: const BorderSide(color: AppColors.brandBrown, width: 1.5),
        ),
        labelStyle: const TextStyle(color: AppColors.text),
        hintStyle: const TextStyle(color: AppColors.textMuted),
      ),
    );
  }
}
