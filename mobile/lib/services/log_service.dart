import 'package:flutter/foundation.dart';
import 'package:path_provider/path_provider.dart';
import 'dart:io';

/// Простой журнал для отладки ISOTOPE
class LogService {
  static final List<String> _logs = [];
  static const int maxLogs = 500;

  static void log(String message) {
    final timestamp = DateTime.now().toIso8601String().substring(11, 23);
    final entry = '[$timestamp] $message';
    _logs.add(entry);
    if (_logs.length > maxLogs) {
      _logs.removeAt(0);
    }
    debugPrint('[ISOTOPE] $entry');
  }

  static List<String> get logs => List.unmodifiable(_logs);

  static String get logsText => _logs.join('\n');

  static void clear() {
    _logs.clear();
  }

  /// Сохраняет журнал в Download
  static Future<String?> saveToFile() async {
    try {
      // Сохраняем в Download — доступно пользователю
      final downloadDir = Directory('/storage/emulated/0/Download');
      if (!downloadDir.existsSync()) {
        downloadDir.createSync(recursive: true);
      }

      final timestamp = DateTime.now()
          .toIso8601String()
          .replaceAll(':', '-')
          .replaceAll('.', '-');
      final file = File('${downloadDir.path}/isotope_log_$timestamp.txt');
      await file.writeAsString(logsText);
      return file.path;
    } catch (e) {
      debugPrint('Save log error: $e');
      return null;
    }
  }
}