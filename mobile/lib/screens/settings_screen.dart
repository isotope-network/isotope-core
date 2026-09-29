// mobile/lib/screens/settings_screen.dart
import 'package:flutter/material.dart';
import 'privacy_settings_screen.dart';
import 'messages_settings_screen.dart';

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
            leading: const Icon(Icons.lock_outline),
            title: const Text('Приватность'),
            subtitle: const Text('Кто видит ваши статусы'),
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