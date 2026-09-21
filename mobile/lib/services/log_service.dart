// mobile/lib/services/log_service.dart
import 'dart:io';
import 'package:flutter/foundation.dart';

class LogService {
  static final List<String> _logs = [];
  static const int _maxLogs = 1000;
  static const int _maxMessageLength = 4096; // 4 КБ — защита от огромных строк
  static File? _logFile;
  static String _logFilePath = '';

  /// Инициализация — загрузка лога из файла
  static Future<void> init(String filesDir) async {
    LogService.log('LogService: init START filesDir=$filesDir');
    _logFilePath = '$filesDir/isotope_app.log';
    _logFile = File(_logFilePath);

    try {
      if (await _logFile!.exists()) {
        final content = await _logFile!.readAsString();
        final lines = content.split('\n');
        _logs.clear();
        for (final l in lines) {
          if (l.isEmpty) continue;
          _logs.add(_truncate(l));
        }
        if (_logs.length > _maxLogs) {
          _logs.removeRange(0, _logs.length - _maxLogs);
        }
        // Перезаписываем файл очищенной и обрезанной версией
        await _logFile!.writeAsString('${_logs.join('\n')}\n');
        debugPrint('LogService: loaded ${_logs.length} logs from file (truncated)');
      } else {
        debugPrint('LogService: no log file found');
      }
    } catch (e) {
      debugPrint('LogService: error loading logs: $e');
    }
    LogService.log('LogService: init END');
  }

  /// Обрезает слишком длинную строку
  static String _truncate(String s) {
    if (s.length <= _maxMessageLength) return s;
    final removed = s.length - _maxMessageLength;
    return '${s.substring(0, _maxMessageLength)}...[truncated $removed chars]';
  }

  /// Добавляет запись в лог
  static void log(String message) {
    final timestamp = _formatTime(DateTime.now());
    final safeMessage = _truncate(message);
    final line = '[$timestamp] $safeMessage';

    _logs.add(line);
    if (_logs.length > _maxLogs) {
      _logs.removeAt(0);
    }

    debugPrint(line);

    // Асинхронно сохраняем в файл
    _saveToFile(line);
  }

  /// Сохраняет строку в файл (асинхронно)
  static Future<void> _saveToFile(String line) async {
    if (_logFile == null) return;

    try {
      await _logFile!.writeAsString('$line\n', mode: FileMode.append);
    } catch (e) {
      debugPrint('LogService: error writing to file: $e');
    }
  }

  /// Форматирует время
  static String _formatTime(DateTime dt) {
    String two(int n) => n.toString().padLeft(2, '0');
    String three(int n) => n.toString().padLeft(3, '0');
    return '${two(dt.hour)}:${two(dt.minute)}:${two(dt.second)}.${three(dt.millisecond)}';
  }

  /// Возвращает все логи
  static List<String> get logs => List.unmodifiable(_logs);

  /// Очищает лог
  static Future<void> clear() async {
    LogService.log('LogService: clear START');
    _logs.clear();
    if (_logFile != null) {
      try {
        await _logFile!.writeAsString('');
        debugPrint('LogService: log cleared');
      } catch (e) {
        debugPrint('LogService: error clearing log: $e');
      }
    }
    LogService.log('LogService: clear END');
  }

  /// Возвращает путь к файлу лога
  static String get logFilePath => _logFilePath;
}