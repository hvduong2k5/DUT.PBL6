import 'package:flutter/material.dart';

/// Design tokens of the "Goi Hue" heritage kit (source: kit `ui.css`).
abstract final class HeritageColors {
  static const kem = Color(0xFFFCF9F4);
  static const kemAm = Color(0xFFF6EFE2);
  static const kemDam = Color(0xFFEDE0CC);
  static const nauDam = Color(0xFF563A26);
  static const nauCungDinh = Color(0xFF352014);
  static const nau = Color(0xFF70513B);
  static const mat = Color(0xFFD4A24C);
  static const matSang = Color(0xFFE6BC6A);
  static const doChuSa = Color(0xFF8B261D);
  static const xanhNgoc = Color(0xFF2D5A3C);
  static const muc = Color(0xFF2F241C);
  static const phu = Color(0xFF655241);
  static const vien = Color(0xFFDFD1BE);
  static const loi = Color(0xFF8F2817);
  static const disabled = Color(0xFFE2D9CC);
}

abstract final class HeritageMotion {
  static const curve = Cubic(0.22, 1, 0.36, 1);
  static const duration = Duration(milliseconds: 240);
}

abstract final class HeritageAssets {
  static const _root = 'assets/heritage';
  static const logo = '$_root/logo-goi-hue.svg';
  static const divider = '$_root/phan-cach.svg';
  static const paper = '$_root/paper.jpg';
  static String motif(String name) => '$_root/motifs/$name.webp';
  static String photo(String name) => '$_root/photos/$name.jpg';
}
