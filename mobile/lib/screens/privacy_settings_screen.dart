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
  bool _maskingEnabled = false;
  String _seedPhrase = '';
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
      final maskingEnabled = await LibP2PService.getMaskingEnabled();
      final seedPhrase = await LibP2PService.getSeedPhrase();
      if (mounted) {
        setState(() {
          _myReadEnabled = readEnabled;
          _showNotificationContent = showContent;
          _notifCount = notifSettings['count'] ?? false;
          _notifNames = notifSettings['names'] ?? false;
          _notifProgress = notifSettings['progress'] ?? false;
          _sphinxEnabled = sphinxEnabled;
          _mixingEnabled = mixingEnabled;
          _maskingEnabled = maskingEnabled;
          _seedPhrase = seedPhrase;
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

  Future<void> _setMaskingEnabled(bool value) async {
    setState(() => _maskingEnabled = value);
    try {
      final result = await LibP2PService.setMaskingEnabled(value);
      if (result.containsKey('error')) {
        LogService.log('PrivacySettings: setMaskingEnabled failed: ${result['error']}');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
          setState(() => _maskingEnabled = !value);
        }
      } else {
        LogService.log('PrivacySettings: masking_enabled=$value');
      }
    } catch (e) {
      LogService.log('PrivacySettings: setMaskingEnabled exception: $e');
      if (mounted) setState(() => _maskingEnabled = !value);
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

  Future<void> _showSeedDialog() async {
    final hasSeed = _seedPhrase.isNotEmpty;
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Seed-фраза'),
        content: Text(
          hasSeed
              ? 'Seed-фраза задана. PeerID восстановим при перезапуске.\n\n'
                  'Восстановить из другой фразы? PeerID изменится, контакты — заново.'
              : 'Seed-фраза не задана. Сейчас используется случайный PeerID (RSA).\n\n'
                  'Создать seed-фразу — PeerID станет восстановимым. '
                  'После перезапуска PeerID изменится, контакты — заново.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('Отмена'),
          ),
          if (!hasSeed)
            TextButton(
              onPressed: () {
                Navigator.of(ctx).pop();
                _createSeed();
              },
              child: const Text('Создать'),
            ),
          if (hasSeed)
            TextButton(
              onPressed: () {
                Navigator.of(ctx).pop();
                _showSeed();
              },
              child: const Text('Показать'),
            ),
          TextButton(
            onPressed: () {
              Navigator.of(ctx).pop();
              _restoreSeed();
            },
            child: const Text('Восстановить'),
          ),
        ],
      ),
    );
  }

  Future<void> _createSeed() async {
    final phrase = await LibP2PService.generateSeedPhrase(12);
    if (phrase.isEmpty) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Не удалось создать фразу')),
        );
      }
      return;
    }
    if (!mounted) return;
    await showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (ctx) => AlertDialog(
        title: const Text('Запишите фразу'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Потеряешь фразу — потеряешь PeerID и контакты. '
              'Никому не показывайте.',
              style: TextStyle(color: Colors.red, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 16),
            SelectableText(
              phrase,
              style: const TextStyle(fontSize: 16, fontFamily: 'monospace'),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('Отмена'),
          ),
          FilledButton(
            onPressed: () async {
              Navigator.of(ctx).pop();
              final result = await LibP2PService.setSeedPhraseBeforeStart(phrase);
              if (!mounted) return;
              if (result.containsKey('error')) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('Ошибка: ${result['error']}')),
                );
                return;
              }
              setState(() => _seedPhrase = phrase);
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('Перезапустите приложение, чтобы применить')),
              );
            },
            child: const Text('Я записал'),
          ),
        ],
      ),
    );
  }

  Future<void> _showSeed() async {
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Ваша seed-фраза'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Никому не показывайте. Кто знает фразу — тот получает ваш PeerID.',
              style: TextStyle(color: Colors.red),
            ),
            const SizedBox(height: 16),
            SelectableText(
              _seedPhrase,
              style: const TextStyle(fontSize: 16, fontFamily: 'monospace'),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('Закрыть'),
          ),
        ],
      ),
    );
  }

  Future<void> _restoreSeed() async {
    var localPhrase = '';
    final phrase = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Восстановить из фразы'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Введите 12 или 24 слова через пробел. '
              'PeerID изменится, контакты — заново.',
              style: TextStyle(color: Colors.red),
            ),
            const SizedBox(height: 16),
            TextField(
              maxLines: 4,
              onChanged: (v) => localPhrase = v,
              decoration: const InputDecoration(
                hintText: 'word1 word2 word3 ...',
                border: OutlineInputBorder(),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(),
            child: const Text('Отмена'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(localPhrase.trim()),
            child: const Text('Восстановить'),
          ),
        ],
      ),
    );
    if (phrase == null || phrase.isEmpty) return;

    final result = await LibP2PService.setSeedPhraseBeforeStart(phrase);
    if (!mounted) return;
    if (result.containsKey('error')) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Ошибка: ${result['error']}')),
      );
      return;
    }
    setState(() => _seedPhrase = phrase);
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Перезапустите приложение, чтобы применить')),
    );
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
                    'Требует 2+ доверенных контакта. '
                    'Файлы — через Sphinx только при включённом Mixing.',
                  ),
                  value: _sphinxEnabled,
                  onChanged: _setSphinxEnabled,
                ),
                SwitchListTile(
                  title: const Text('Mixing'),
                  subtitle: Text(
                    _sphinxEnabled
                        ? 'Скрывает тайминги. Текст: +13–15 сек. '
                          'Файлы: ~2.7 мин на 1 МБ. '
                          'Тип сообщения виден relay. Полное скрытие — v2.0.4.'
                        : 'Доступно только при включённом Sphinx.',
                  ),
                  value: _mixingEnabled,
                  onChanged: _sphinxEnabled ? _setMixingEnabled : null,
                ),
                SwitchListTile(
                  title: const Text('Маскировка'),
                  subtitle: const Text(
                    'Dummy-трафик: пустой пакет каждые 60 сек. '
                    'Размер 64 КБ — всегда (Sphinx). Трафик ×1.4. '
                    'Тип, объём, паттерн скрыты.',
                  ),
                  value: _maskingEnabled,
                  onChanged: _setMaskingEnabled,
                ),
                const Divider(),
                const Padding(
                  padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                  child: Text(
                    'Seed-фраза',
                    style: TextStyle(fontWeight: FontWeight.bold),
                  ),
                ),
                const Padding(
                  padding: EdgeInsets.symmetric(horizontal: 16),
                  child: Text(
                    'Восстановление PeerID. Если потеряете устройство — '
                    'введёте фразу на новом, PeerID вернётся. Контакты — заново (QR).',
                    style: TextStyle(fontSize: 12, color: Colors.grey),
                  ),
                ),
                ListTile(
                  leading: Icon(
                    _seedPhrase.isEmpty ? Icons.key_off : Icons.key,
                    color: _seedPhrase.isEmpty ? Colors.grey : Colors.green,
                  ),
                  title: Text(
                    _seedPhrase.isEmpty ? 'Seed-фраза не задана' : 'Seed-фраза задана',
                  ),
                  subtitle: Text(
                    _seedPhrase.isEmpty
                        ? 'Создать или восстановить'
                        : 'Показать, восстановить из другой',
                  ),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: _showSeedDialog,
                ),
              ],
            ),
    );
  }
}
// mobile/lib/screens/privacy_settings_screen.dart