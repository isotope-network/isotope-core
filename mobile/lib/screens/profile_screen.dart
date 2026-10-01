// mobile/lib/screens/profile_screen.dart
import 'package:flutter/material.dart';
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
            SnackBar(content: Text('Ошибка: ${result['error']}')),
          );
        }
      } else {
        LogService.log('ProfileSettings: my_display_name="$name"');
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Имя сохранено')),
          );
        }
      }
    } catch (e) {
      LogService.log('ProfileSettings: save exception: $e');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Ошибка: $e')),
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
        title: const Text('Профиль'),
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                const Text(
                  'Ваше имя',
                  style: TextStyle(fontSize: 14, fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 8),
                TextField(
                  controller: _controller,
                  maxLength: 100,
                  enabled: !_saving,
                  decoration: const InputDecoration(
                    hintText: 'Например: Иван',
                    border: OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 8),
                const Text(
                  'Это представление по умолчанию.\n'
                  'Используется в QR и запросах.\n'
                  'Можно не задавать. Тогда контакты увидят только ваш код.',
                  style: TextStyle(fontSize: 12, color: Colors.grey),
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
                      : const Text('Сохранить'),
                ),
              ],
            ),
    );
  }
}