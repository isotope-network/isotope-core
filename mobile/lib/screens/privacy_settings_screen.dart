// mobile/lib/screens/privacy_settings_screen.dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/chat_provider.dart';
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
  bool _notifCount = false;
  bool _notifNames = false;
  bool _notifProgress = false;
  bool _sphinxEnabled = false;
  bool _mixingEnabled = false;
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
      final notifSettings = await LibP2PService.getNotificationSettings();
      final sphinxEnabled = await LibP2PService.getSphinxEnabled();
      final mixingEnabled = await LibP2PService.getMixingEnabled();
      if (mounted) {
        setState(() {
          _myReadEnabled = readEnabled;
          _showNotificationContent = showContent;
          _notifCount = notifSettings['count'] ?? false;
          _notifNames = notifSettings['names'] ?? false;
          _notifProgress = notifSettings['progress'] ?? false;
          _sphinxEnabled = sphinxEnabled;
          _mixingEnabled = mixingEnabled;
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

  Future<void> _setNotifSetting(String key, bool value) async {
    // Оптимистичное обновление.
    final prev = {
      'count': _notifCount,
      'names': _notifNames,
      'progress': _notifProgress,
    };
    setState(() {
      switch (key) {
        case 'count':
          _notifCount = value;
          break;
        case 'names':
          _notifNames = value;
          break;
        case 'progress':
          _notifProgress = value;
          break;
      }
    });
    try {
      final settings = {
        'count': _notifCount,
        'names': _notifNames,
        'progress': _notifProgress,
      };
      final result = await LibP2PService.setNotificationSettings(settings);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: setNotifSetting failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          // Откат.
          setState(() {
            _notifCount = prev['count']!;
            _notifNames = prev['names']!;
            _notifProgress = prev['progress']!;
          });
        }
      } else {
        LogService.log('PrivacySettings: $key=$value');
        // Перечитать настройки в ChatProvider — шторка обновится сразу.
        if (mounted) {
          try {
            await context.read<ChatProvider>().reloadNotificationSettings();
          } catch (_) {}
        }
      }
    } catch (e) {
      LogService.log('PrivacySettings: setNotifSetting exception: $e');
      if (mounted) {
        setState(() {
          _notifCount = prev['count']!;
          _notifNames = prev['names']!;
          _notifProgress = prev['progress']!;
        });
      }
    }
  }

  Future<void> _setSphinxEnabled(bool value) async {
    setState(() => _sphinxEnabled = value);
    try {
      final result = await LibP2PService.setSphinxEnabled(value);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: setSphinxEnabled failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          setState(() => _sphinxEnabled = !value);
        }
      } else {
        LogService.log('PrivacySettings: sphinx_enabled=$value');
      }
    } catch (e) {
      LogService.log('PrivacySettings: setSphinxEnabled exception: $e');
      if (mounted) setState(() => _sphinxEnabled = !value);
    }
  }

  Future<void> _setMixingEnabled(bool value) async {
    setState(() => _mixingEnabled = value);
    try {
      final result = await LibP2PService.setMixingEnabled(value);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: setMixingEnabled failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          setState(() => _mixingEnabled = !value);
        }
      } else {
        LogService.log('PrivacySettings: mixing_enabled=$value');
      }
    } catch (e) {
      LogService.log('PrivacySettings: setMixingEnabled exception: $e');
      if (mounted) setState(() => _mixingEnabled = !value);
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
                const Divider(),
                const Padding(
                  padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                  child: Text(
                    'Уведомление «Сеть активна»',
                    style: TextStyle(fontWeight: FontWeight.bold),
                  ),
                ),
                const Padding(
                  padding: EdgeInsets.symmetric(horizontal: 16),
                  child: Text(
                    'Что показывать в постоянном уведомлении, пока приложение в фоне.',
                    style: TextStyle(fontSize: 12, color: Colors.grey),
                  ),
                ),
                SwitchListTile(
                  title: const Text('Показывать количество'),
                  subtitle: const Text('«3 непрочитанных»'),
                  value: _notifCount,
                  onChanged: (v) => _setNotifSetting('count', v),
                ),
                SwitchListTile(
                  title: const Text('Показывать имена контактов'),
                  subtitle: const Text('«Иван (2), Пётр (1)»'),
                  value: _notifNames,
                  onChanged: (v) => _setNotifSetting('names', v),
                ),
                SwitchListTile(
                  title: const Text('Показывать прогресс файлов'),
                  subtitle: const Text('«📎 file.pdf 45%» + прогресс-бар'),
                  value: _notifProgress,
                  onChanged: (v) => _setNotifSetting('progress', v),
                ),
                const Divider(),
                const Padding(
                  padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                  child: Text(
                    'Анонимность',
                    style: TextStyle(fontWeight: FontWeight.bold),
                  ),
                ),
                SwitchListTile(
                  title: const Text('Sphinx-маршрутизация'),
                  subtitle: const Text(
                    'Сообщения идут через 2 relay + получатель. '
                    'Relay не видит, кто с кем. Получатель видит отправителя. '
                    'Требует 2+ доверенных контакта. Файлы — напрямую.',
                  ),
                  value: _sphinxEnabled,
                  onChanged: _setSphinxEnabled,
                ),
                SwitchListTile(
                  title: const Text('Mixing'),
                  subtitle: Text(
                    _sphinxEnabled
                        ? 'Скрывает тайминги. Ваши сообщения идут с задержкой 13–15 сек. Работает через relay.'
                        : 'Доступно только при включённом Sphinx.',
                  ),
                  value: _mixingEnabled,
                  onChanged: _sphinxEnabled ? _setMixingEnabled : null,
                ),
              ],
            ),
    );
  }
}
// mobile/lib/screens/privacy_settings_screen.dart