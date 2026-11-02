// mobile/lib/screens/settings_screen.dart
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../providers/locale_provider.dart';
import 'privacy_settings_screen.dart';
import 'messages_settings_screen.dart';
import 'profile_screen.dart';

/// Экран «Настройки» — список разделов.
/// Личные — выше Системных (пользовательское важнее).
class SettingsScreen extends StatelessWidget {
  /// Колбэк «Подключение» — открывает диалог bootstrap.
  /// Передаётся из ConnectScreen (там — контроллеры и логика).
  final VoidCallback? onConnection;

  /// Колбэк «Ввести код контакта» — открывает диалог ручного ввода.
  final VoidCallback? onManualCode;

  const SettingsScreen({
    super.key,
    this.onConnection,
    this.onManualCode,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Настройки'),
      ),
      body: ListView(
        children: [
          // ==== ЛИЧНЫЕ ====
          const _SectionHeader('Личные'),
          ListTile(
            leading: const Icon(Icons.person_outline),
            title: const Text('Профиль'),
            subtitle: const Text('Ваше имя'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              Navigator.push(
                context,
                MaterialPageRoute(builder: (_) => const ProfileScreen()),
              );
            },
          ),
          ListTile(
            leading: const Icon(Icons.lock_outline),
            title: const Text('Приватность'),
            subtitle: const Text('Статусы, уведомления, анонимность, seed-фраза'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              Navigator.push(
                context,
                MaterialPageRoute(builder: (_) => const PrivacySettingsScreen()),
              );
            },
          ),
          ListTile(
            leading: const Icon(Icons.message_outlined),
            title: const Text('Сообщения'),
            subtitle: const Text('Задержка, время удаления'),
            trailing: const Icon(Icons.chevron_right),
            onTap: () {
              Navigator.push(
                context,
                MaterialPageRoute(builder: (_) => const MessagesSettingsScreen()),
              );
            },
          ),

          const Divider(height: 1),

          // ==== СИСТЕМНЫЕ ====
          const _SectionHeader('Системные'),
          ListTile(
            leading: const Icon(Icons.language),
            title: const Text('Язык'),
            subtitle: Text(_currentLanguageName(context)),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => _showLanguageDialog(context),
          ),
          ListTile(
            leading: const Icon(Icons.link),
            title: const Text('Подключение'),
            subtitle: const Text('Адрес подключения'),
            enabled: onConnection != null,
            onTap: onConnection,
          ),
          ListTile(
            leading: const Icon(Icons.input),
            title: const Text('Ввести код контакта'),
            subtitle: const Text('Для продвинутых'),
            enabled: onManualCode != null,
            onTap: onManualCode,
          ),

          const SizedBox(height: 16),
        ],
      ),
    );
  }

  String _currentLanguageName(BuildContext context) {
    final provider = context.watch<LocaleProvider>();
    switch (provider.code) {
      case 'en':
        return 'English';
      case 'ru':
        return 'Русский';
      default:
        return 'Системный';
    }
  }

  void _showLanguageDialog(BuildContext context) {
    final provider = context.read<LocaleProvider>();
    showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Язык'),
        content: RadioGroup<String>(
          groupValue: provider.code,
          onChanged: (v) {
            if (v == null) return;
            provider.setLocale(v);
            Navigator.of(ctx).pop();
          },
          child: const Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              RadioListTile<String>(
                title: Text('Системный'),
                value: '',
              ),
              RadioListTile<String>(
                title: Text('English'),
                value: 'en',
              ),
              RadioListTile<String>(
                title: Text('Русский'),
                value: 'ru',
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Заголовок раздела.
class _SectionHeader extends StatelessWidget {
  final String title;
  const _SectionHeader(this.title);

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
      child: Text(
        title.toUpperCase(),
        style: TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.bold,
          color: Colors.grey.shade600,
          letterSpacing: 0.5,
        ),
      ),
    );
  }
}
// mobile/lib/screens/settings_screen.dart