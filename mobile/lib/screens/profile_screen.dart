// mobile/lib/screens/profile_screen.dart
import 'package:flutter/material.dart';
import '../l10n/app_localizations.dart';
import '../services/libp2p_service.dart';
import '../services/log_service.dart';

/// Экран «Профиль».
/// Поле «Ваше имя» — представление по умолчанию.
/// Используется в QR и [CONTACT_REQUEST], если не переопределено.
class ProfileScreen extends StatefulWidget {
  const ProfileScreen({super.key});

  @override
  State<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends State<ProfileScreen> {
  final TextEditingController _controller = TextEditingController();
  bool _loading = true;
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final name = await LibP2PService.getMyDisplayName();
      if (mounted) {
        setState(() {
          _controller.text = name;
          _loading = false;
        });
      }
    } catch (e) {
      LogService.log('ProfileSettings: load failed: $e');
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _save() async {
    final name = _controller.text.trim();
    setState(() => _saving = true);
    try {
      final result = await LibP2PService.setMyDisplayName(name);
      if (result.containsKey('error')) {
        LogService.log('ProfileSettings: save failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('${AppLocalizations.of(context).error}: ${result['error']}')),
          );
        }
      } else {
        LogService.log('ProfileSettings: my_display_name="$name"');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(AppLocalizations.of(context).profileNameSaved)),
          );
        }
      }
    } catch (e) {
      LogService.log('ProfileSettings: save exception: $e');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('${AppLocalizations.of(context).error}: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(AppLocalizations.of(context).profileTitle),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                Text(
                  AppLocalizations.of(context).profileYourName,
                  style: const TextStyle(fontSize: 14, fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 8),
                TextField(
                  controller: _controller,
                  maxLength: 100,
                  enabled: !_saving,
                  decoration: InputDecoration(
                    hintText: AppLocalizations.of(context).profileNameHint,
                    border: const OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  AppLocalizations.of(context).profileNameSub,
                  style: const TextStyle(fontSize: 12, color: Colors.grey),
                ),
                const SizedBox(height: 24),
                FilledButton(
                  onPressed: _saving ? null : _save,
                  child: _saving
                      ? const SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: Colors.white,
                          ),
                        )
                      : Text(AppLocalizations.of(context).save),
                ),
              ],
            ),
    );
  }
}
// mobile/lib/screens/profile_screen.dart