// mobile/lib/utils/time_format.dart
//
// Умный формат времени для сообщений и списка контактов.
// Правила:
//   Сегодня  — "14:13"
//   Вчера    — "Вчера, 14:13"
//   Раньше   — "25.09.26, 14:13"
//
// Плюс — разделитель дат для чата:
//   Сегодня  — "Сегодня"
//   Вчера    — "Вчера"
//   Раньше   — "25 сентября"

/// Парсит ISO-8601 строку в локальное время.
/// Go присылает UTC без суффикса Z — добавляем.
DateTime? parseIsoLocal(String iso) {
  if (iso.isEmpty) return null;
  var t = iso.trim();
  if (!t.endsWith('Z') && !t.contains('+')) {
    // Проверяем, есть ли минус-смещение (не считаем первый минус даты).
    final tIndex = t.indexOf('T');
    if (tIndex >= 0) {
      final after = t.substring(tIndex);
      if (!after.contains('-')) {
        t = '${t}Z';
      }
    }
  }
  final dt = DateTime.tryParse(t);
  return dt?.toLocal();
}

/// Форматирует время сообщения для отображения.
///   Сегодня  — "14:13"
///   Вчера    — "Вчера, 14:13"
///   Раньше   — "25.09.26, 14:13"
String formatMessageTime(String iso) {
  final dt = parseIsoLocal(iso);
  if (dt == null) return iso;

  final now = DateTime.now();
  final today = DateTime(now.year, now.month, now.day);
  final msgDay = DateTime(dt.year, dt.month, dt.day);
  final diffDays = today.difference(msgDay).inDays;

  final hh = dt.hour.toString().padLeft(2, '0');
  final mm = dt.minute.toString().padLeft(2, '0');

  if (diffDays == 0) {
    return '$hh:$mm';
  }
  if (diffDays == 1) {
    return 'Вчера, $hh:$mm';
  }
  final dd = dt.day.toString().padLeft(2, '0');
  final mo = dt.month.toString().padLeft(2, '0');
  final yy = (dt.year % 100).toString().padLeft(2, '0');
  return '$dd.$mo.$yy, $hh:$mm';
}

/// Короткий формат времени для списка контактов (без слов).
///   Сегодня  — "14:13"
///   Вчера    — "Вчера"
///   Раньше   — "25.09"
String formatShortTime(String iso) {
  final dt = parseIsoLocal(iso);
  if (dt == null) return '';

  final now = DateTime.now();
  final today = DateTime(now.year, now.month, now.day);
  final msgDay = DateTime(dt.year, dt.month, dt.day);
  final diffDays = today.difference(msgDay).inDays;

  final hh = dt.hour.toString().padLeft(2, '0');
  final mm = dt.minute.toString().padLeft(2, '0');

  if (diffDays == 0) return '$hh:$mm';
  if (diffDays == 1) return 'Вчера';

  final dd = dt.day.toString().padLeft(2, '0');
  final mo = dt.month.toString().padLeft(2, '0');
  return '$dd.$mo';
}

/// Возвращает метку-разделитель дат для чата.
///   Сегодня  — "Сегодня"
///   Вчера    — "Вчера"
///   Раньше   — "25 сентября"
String formatDateSeparator(DateTime dt) {
  final now = DateTime.now();
  final today = DateTime(now.year, now.month, now.day);
  final day = DateTime(dt.year, dt.month, dt.day);
  final diffDays = today.difference(day).inDays;

  if (diffDays == 0) return 'Сегодня';
  if (diffDays == 1) return 'Вчера';

  const months = [
    '', // placeholder, январь = 1
    'января',
    'февраля',
    'марта',
    'апреля',
    'мая',
    'июня',
    'июля',
    'августа',
    'сентября',
    'октября',
    'ноября',
    'декабря',
  ];
  return '${dt.day} ${months[dt.month]}';
}