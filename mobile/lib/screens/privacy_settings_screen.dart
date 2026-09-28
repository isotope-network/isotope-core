// mobile/lib/screens/privacy_settings_screen.dart
import 'package:flutter/material.dart';
import '../services/libp2p_service.dart';
import '../services/log_service.dart';

/// Экран «Приватность».
/// Пока один тумблер: «Показывать статус прочтения».
class PrivacySettingsScreen extends StatefulWidget {
  const PrivacySettingsScreen({super.key});

  @override
  State<PrivacySettingsScreen> createState() => _PrivacySettingsScreenState();
}

class _PrivacySettingsScreenState extends State<PrivacySettingsScreen> {
  bool _myReadEnabled = true;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final enabled = await LibP2PService.getMyReadEnabled();
      if (mounted) {
        setState(() {
          _myReadEnabled = enabled;
          _loading = false;
        });
      }
    } catch (e) {
      LogService.log('PrivacySettings: load failed: $e');
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _setEnabled(bool value) async {
    setState(() => _myReadEnabled = value);
    try {
      final result = await LibP2PService.setMyReadEnabled(value);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: set failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          // Откат.
          setState(() => _myReadEnabled = !value);
        }
      } else {
        LogService.log('PrivacySettings: my_read_enabled=$value');
      }
    } catch (e) {
      LogService.log('PrivacySettings: set exception: $e');
      if (mounted) setState(() => _myReadEnabled = !value);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Приватность'),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              children: [
                SwitchListTile(
                  title: const Text('Показывать статус прочтения'),
                  subtitle: const Text(
                    'Когда включено — вы видите, прочитал ли собеседник ваше сообщение, '
                    'и он видит ваш статус. Когда выключено — оба скрыты.',
                  ),
                  value: _myReadEnabled,
                  onChanged: _setEnabled,
                ),
              ],
            ),
    );
  }
}
// mobile/lib/screens/privacy_settings_screen.dart