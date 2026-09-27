// mobile/lib/screens/onboarding_screen.dart
import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'connect_screen.dart';

/// Ключ флага «onboarding пройден». Должен совпадать с main.dart.
const String FIRST_LAUNCH_KEY = 'first_launch_seen';

/// Экран первого запуска — 3 страницы.
///
/// Экран 1 — знакомство.
/// Экран 2 — ваш код готов.
/// Экран 3 — добавьте первый контакт (с действиями).
///
/// После 3-го экрана или «Пропустить» — флаг в SharedPreferences
/// и переход в ConnectScreen. Опционально — initialAction ('scan' / 'showQR').
class OnboardingScreen extends StatefulWidget {
  const OnboardingScreen({super.key});

  @override
  State<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _OnboardingScreenState extends State<OnboardingScreen> {
  final PageController _controller = PageController();
  int _page = 0;

  static const int _lastPage = 2;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _next() {
    if (_page < _lastPage) {
      _controller.nextPage(
        duration: const Duration(milliseconds: 250),
        curve: Curves.easeOut,
      );
    }
  }

  Future<void> _finish({String? action}) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setBool(FIRST_LAUNCH_KEY, true);
    } catch (_) {}

    if (!mounted) return;

    Navigator.of(context).pushReplacement(
      MaterialPageRoute(
        builder: (_) => ConnectScreen(initialAction: action),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            Expanded(
              child: PageView(
                controller: _controller,
                onPageChanged: (i) => setState(() => _page = i),
                children: [
                  _Page1(),
                  _Page2(),
                  _Page3(onScan: () => _finish(action: 'scan'), onShowQR: () => _finish(action: 'showQR'), onSkip: () => _finish()),
                ],
              ),
            ),
            // Точки-индикатор внизу.
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: List.generate(3, (i) {
                  final active = i == _page;
                  return AnimatedContainer(
                    duration: const Duration(milliseconds: 200),
                    margin: const EdgeInsets.symmetric(horizontal: 4),
                    width: active ? 20 : 8,
                    height: 8,
                    decoration: BoxDecoration(
                      color: active ? Colors.green : Colors.grey.shade300,
                      borderRadius: BorderRadius.circular(4),
                    ),
                  );
                }),
              ),
            ),
            // Кнопка «Далее» — на экранах 1 и 2.
            if (_page < _lastPage)
              Padding(
                padding: const EdgeInsets.fromLTRB(24, 0, 24, 24),
                child: SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: _next,
                    style: FilledButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 16),
                    ),
                    child: const Text('Далее'),
                  ),
                ),
              )
            else
              const SizedBox(height: 24),
          ],
        ),
      ),
    );
  }
}

// ============================================================
// ЭКРАН 1 — знакомство.
// ============================================================

class _Page1 extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 32),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          // Логотип — зелёная иконка с локальным стилем.
          Container(
            width: 120,
            height: 120,
            decoration: BoxDecoration(
              color: Colors.green.shade50,
              borderRadius: BorderRadius.circular(28),
            ),
            child: Icon(
              Icons.shield_outlined,
              size: 72,
              color: Colors.green.shade700,
            ),
          ),
          const SizedBox(height: 32),
          const Text(
            'Добро пожаловать в ISOTOPE',
            textAlign: TextAlign.center,
            style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 16),
          Text(
            'Общайтесь без цензуры и слежки.\n'
            'Сообщения защищены. Никто не видит,\n'
            'кто вы и что вы пишете.',
            textAlign: TextAlign.center,
            style: TextStyle(fontSize: 15, color: Colors.grey.shade700, height: 1.5),
          ),
        ],
      ),
    );
  }
}

// ============================================================
// ЭКРАН 2 — ваш код готов.
// ============================================================

class _Page2 extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 32),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Container(
            width: 120,
            height: 120,
            decoration: BoxDecoration(
              color: Colors.green.shade50,
              borderRadius: BorderRadius.circular(28),
            ),
            child: Icon(
              Icons.qr_code_2,
              size: 72,
              color: Colors.green.shade700,
            ),
          ),
          const SizedBox(height: 32),
          const Text(
            'Ваш код готов',
            textAlign: TextAlign.center,
            style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 16),
          Text(
            'Мы создали для вас уникальный код.\n'
            'Он нужен, чтобы друзья могли добавить\n'
            'вас в контакты.',
            textAlign: TextAlign.center,
            style: TextStyle(fontSize: 15, color: Colors.grey.shade700, height: 1.5),
          ),
        ],
      ),
    );
  }
}

// ============================================================
// ЭКРАН 3 — добавьте первый контакт.
// ============================================================

class _Page3 extends StatelessWidget {
  final VoidCallback onScan;
  final VoidCallback onShowQR;
  final VoidCallback onSkip;

  const _Page3({
    required this.onScan,
    required this.onShowQR,
    required this.onSkip,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 32),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Container(
            width: 120,
            height: 120,
            decoration: BoxDecoration(
              color: Colors.green.shade50,
              borderRadius: BorderRadius.circular(28),
            ),
            child: Icon(
              Icons.person_add_alt_1_outlined,
              size: 72,
              color: Colors.green.shade700,
            ),
          ),
          const SizedBox(height: 32),
          const Text(
            'Добавьте первый контакт',
            textAlign: TextAlign.center,
            style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 16),
          Text(
            'Отсканируйте QR друга или покажите\n'
            'свой — чтобы начать общение.',
            textAlign: TextAlign.center,
            style: TextStyle(fontSize: 15, color: Colors.grey.shade700, height: 1.5),
          ),
          const SizedBox(height: 32),
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              onPressed: onScan,
              icon: const Icon(Icons.qr_code_scanner),
              label: const Text('Сканировать'),
              style: FilledButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
            ),
          ),
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: OutlinedButton.icon(
              onPressed: onShowQR,
              icon: const Icon(Icons.qr_code),
              label: const Text('Показать мой QR'),
              style: OutlinedButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
            ),
          ),
          const SizedBox(height: 12),
          TextButton(
            onPressed: onSkip,
            child: Text(
              'Пропустить',
              style: TextStyle(color: Colors.grey.shade600),
            ),
          ),
        ],
      ),
    );
  }
}
// mobile/lib/screens/onboarding_screen.dart