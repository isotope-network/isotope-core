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
  bool _showNotificationContent = true;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final readEnabled = await LibP2PService.getMyReadEnabled();
      final showContent = await LibP2PService.getShowNotificationContent();
      if (mounted) {
        setState(() {
          _myReadEnabled = readEnabled;
          _showNotificationContent = showContent;
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

  Future<void> _setShowContent(bool value) async {
    setState(() => _showNotificationContent = value);
    try {
      final result = await LibP2PService.setShowNotificationContent(value);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: setShowContent failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          setState(() => _showNotificationContent = !value);
        }
      } else {
        LogService.log('PrivacySettings: show_notification_content=$value');
      }
    } catch (e) {
      LogService.log('PrivacySettings: setShowContent exception: $e');
      if (mounted) setState(() => _showNotificationContent = !value);
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
                SwitchListTile(
                  title: const Text('Показывать содержимое уведомлений'),
                  subtitle: const Text(
                    'Когда включено — в уведомлении отображается имя контакта и начало сообщения. '
                    'Когда выключено — только «Новое сообщение».',
                  ),
                  value: _showNotificationContent,
                  onChanged: _setShowContent,
                ),
              ],
            ),
    );
  }
}
// mobile/lib/screens/privacy_settings_screen.dart