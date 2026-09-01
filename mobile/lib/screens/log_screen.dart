import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import '../services/log_service.dart';

class LogScreen extends StatelessWidget {
  const LogScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final logs = LogService.logs;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Журнал ISOTOPE'),
        actions: [
          IconButton(
            icon: const Icon(Icons.save),
            onPressed: () async {
              const channel = MethodChannel('isotope/libp2p');
              final result = await channel.invokeMethod('saveLog', {
                'text': LogService.logsText,
              });
              if (context.mounted && result != null) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('$result')),
                );
              }
            },
            tooltip: 'Сохранить',
          ),
          IconButton(
            icon: const Icon(Icons.delete_outline),
            onPressed: () {
              LogService.clear();
              if (context.mounted) {
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Журнал очищен')),
                );
              }
            },
            tooltip: 'Очистить',
          ),
        ],
      ),
      body: logs.isEmpty
          ? const Center(child: Text('Журнал пуст'))
          : SingleChildScrollView(
              child: SelectableText(
                logs.reversed.join('\n'),
                style: const TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 11,
                ),
              ),
            ),
    );
  }
}