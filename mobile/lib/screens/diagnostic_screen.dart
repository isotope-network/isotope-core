import 'package:flutter/material.dart';
import 'dart:async';
import 'dart:convert';
import 'package:http/http.dart' as http;

class DiagnosticScreen extends StatefulWidget {
  final String nodeAddress;

  const DiagnosticScreen({super.key, required this.nodeAddress});

  @override
  State<DiagnosticScreen> createState() => _DiagnosticScreenState();
}

class _DiagnosticScreenState extends State<DiagnosticScreen> {
  final List<Map<String, String>> _results = [];
  bool _running = false;

  @override
  void initState() {
    super.initState();
    _runDiagnostics();
  }

  Future<void> _runDiagnostics() async {
    setState(() {
      _running = true;
      _results.clear();
    });

    final cleanAddress = widget.nodeAddress.replaceFirst('http://', '');
    final base = 'http://$cleanAddress';

    await _checkStep('1. HTTP GET /status', () async {
      final r = await http.get(Uri.parse('$base/status'))
          .timeout(const Duration(seconds: 10));
      return r.statusCode == 200
          ? '✅ OK\nОтвет: ${r.body}'
          : '❌ HTTP ${r.statusCode}';
    });

    await _checkStep('2. HTTP GET /messages', () async {
      final r = await http.get(Uri.parse('$base/messages'))
          .timeout(const Duration(seconds: 10));
      if (r.statusCode != 200) return '❌ HTTP ${r.statusCode}';
      final body = json.decode(r.body);
      return '✅ OK\nТип: ${body.runtimeType}\nДлина: ${body.toString().length}';
    });

    await _checkStep('3. HTTP GET /channels', () async {
      final r = await http.get(Uri.parse('$base/channels'))
          .timeout(const Duration(seconds: 10));
      if (r.statusCode != 200) return '❌ HTTP ${r.statusCode}';
      return '✅ OK\nОтвет: ${r.body}';
    });

    await _checkStep('4. WebSocket /ws', () async {
      final wsUrl = 'ws://$cleanAddress/ws';
      return 'ℹ️ URL: $wsUrl\n(тест WebSocket в разработке)';
    });

    if (mounted) setState(() => _running = false);
  }

  Future<void> _checkStep(String name, Future<String> Function() fn) async {
    setState(() {
      _results.add({'name': name, 'result': '⏳ Выполняется...'});
    });

    String result;
    try {
      result = await fn();
    } catch (e) {
      result = '❌ Ошибка: $e';
    }

    setState(() {
      _results.removeLast();
      _results.add({'name': name, 'result': result});
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('Диагностика ${widget.nodeAddress}')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            ElevatedButton(
              onPressed: _running ? null : _runDiagnostics,
              child: Text(_running ? 'Проверяю...' : 'Запустить заново'),
            ),
            const SizedBox(height: 16),
            Expanded(
              child: ListView.builder(
                itemCount: _results.length,
                itemBuilder: (_, i) => Card(
                  child: Padding(
                    padding: const EdgeInsets.all(12),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          _results[i]['name']!,
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                        const SizedBox(height: 4),
                        Text(_results[i]['result']!),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}