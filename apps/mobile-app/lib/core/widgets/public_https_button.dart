import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

Uri? publicHttpsUri(String value) {
  final uri = Uri.tryParse(value);
  if (uri == null ||
      uri.scheme != 'https' ||
      uri.host.isEmpty ||
      uri.userInfo.isNotEmpty ||
      uri.host == 'localhost' ||
      uri.host == '127.0.0.1' ||
      uri.host == '::1') {
    return null;
  }
  return uri;
}

class PublicHttpsButton extends StatefulWidget {
  const PublicHttpsButton({super.key, required this.url, required this.label});
  final String url, label;
  @override
  State<PublicHttpsButton> createState() => _PublicHttpsButtonState();
}

class _PublicHttpsButtonState extends State<PublicHttpsButton> {
  bool _pending = false;
  Future<void> _open() async {
    if (_pending) return;
    final uri = publicHttpsUri(widget.url);
    if (uri == null) return;
    setState(() => _pending = true);
    var opened = false;
    try {
      opened = await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (_) {
      /* Recover below without logging potentially sensitive URLs. */
    }
    if (!mounted) return;
    setState(() => _pending = false);
    if (!opened) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text(
            'Chưa thể mở liên kết trên thiết bị. Vui lòng thử lại.',
          ),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) => OutlinedButton.icon(
    onPressed: _pending || publicHttpsUri(widget.url) == null ? null : _open,
    icon: const Icon(Icons.open_in_new),
    label: Text(_pending ? 'Đang mở…' : widget.label),
  );
}
