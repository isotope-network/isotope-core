// D:\isotope\mobile\lib\providers\locale_provider.dart
import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// LocaleProvider — хранит выбранный язык (en/ru/системный).
/// null — системный (MaterialApp сам выберет из supportedLocales).
/// 'en' / 'ru' — явный выбор. Хранится в SharedPreferences (app_locale).
class LocaleProvider extends ChangeNotifier {
  static const String prefsKey = 'app_locale';

  Locale? _locale;
  bool _loaded = false;

  Locale? get locale => _locale;
  bool get loaded => _loaded;

  /// Загружает язык из SharedPreferences. Вызывать при старте.
  Future<void> load() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final code = prefs.getString(prefsKey) ?? '';
      if (code == 'en' || code == 'ru') {
        _locale = Locale(code);
      } else {
        _locale = null;
      }
    } catch (_) {
      _locale = null;
    }
    _loaded = true;
    notifyListeners();
  }

  /// Устанавливает язык. code: 'en' | 'ru' | '' (системный).
  Future<void> setLocale(String code) async {
    if (code == 'en' || code == 'ru') {
      _locale = Locale(code);
    } else {
      _locale = null;
    }
    notifyListeners();
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setString(prefsKey, code);
    } catch (_) {}
  }

  /// Возвращает код текущего языка: 'en' | 'ru' | '' (системный).
  String get code {
    if (_locale == null) return '';
    return _locale!.languageCode;
  }
}