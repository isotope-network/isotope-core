import 'dart:convert';
import 'package:crypto/crypto.dart';

/// Этическая фильтрация ISOTOPE
class EthicsService {
  // Семь универсальных заповедей
  static const String _commandments = 'Не причиняй вреда. '
      'Не кради. '
      'Не обманывай. '
      'Не предавай. '
      'Не будь равнодушным. '
      'Созидай, а не разрушай. '
      'Относись к другим как к себе.';

  static const int vectorDim = 100;
  static const double minWeight = 0.3;

  /// Этический эталон — 100-мерный вектор
  static final List<double> ethicsVector = _computeEthicsVector();

  /// Этический хеш — строка для передачи в ядро
  static final String ethicsHash = _hashText(_commandments);

  /// Порог для блокировки
  static const double blockThreshold = 0.3;

  /// Вычисляет этический вектор из заповедей
  static List<double> _computeEthicsVector() {
    final hash = _hashText(_commandments);
    return _hashToVector(hash);
  }

  /// SHA-256 от текста — возвращает hex-строку
  static String _hashText(String text) {
    final bytes = utf8.encode(text);
    final digest = sha256.convert(bytes);
    return digest.toString();
  }

  /// Хеш → 100-мерный вектор
  static List<double> _hashToVector(String hash) {
    final vec = List<double>.filled(vectorDim, 0);
    for (int i = 0; i < hash.length && i < vectorDim; i++) {
      vec[i] = hash.codeUnitAt(i) / 255.0;
    }
    return vec;
  }

  /// Текст → 100-мерный вектор (символы 30%, биграммы 70%)
  static List<double> textToVector(String text) {
    final vec = List<double>.filled(vectorDim, 0);
    final lower = text.toLowerCase();

    // Символы — 30%
    for (int i = 0; i < lower.length; i++) {
      vec[i % vectorDim] += lower.codeUnitAt(i) * 0.3;
    }

    // Биграммы — 70%
    for (int i = 0; i < lower.length - 1; i++) {
      final bigram = lower.substring(i, i + 2);
      final h = _hashText(bigram);
      for (int j = 0; j < vectorDim; j++) {
        vec[j] += h.codeUnitAt(j % h.length) / 255.0 * 0.7;
      }
    }

    // Нормализация
    var max = 0.0;
    for (final v in vec) {
      if (v > max) max = v;
    }
    if (max > 0) {
      for (int i = 0; i < vec.length; i++) {
        vec[i] = vec[i] / max;
        if (vec[i] > 1.0) vec[i] = 1.0;
      }
    }

    return vec;
  }

  /// Косинусное сходство
  static double cosineSimilarity(List<double> a, List<double> b) {
    if (a.length != b.length || a.isEmpty) return 0;

    var dot = 0.0;
    var normA = 0.0;
    var normB = 0.0;

    for (int i = 0; i < a.length; i++) {
      dot += a[i] * b[i];
      normA += a[i] * a[i];
      normB += b[i] * b[i];
    }

    if (normA == 0 || normB == 0) return 0;
    return dot / (_sqrt(normA) * _sqrt(normB));
  }

  /// Квадратный корень
  static double _sqrt(double x) {
    if (x <= 0) return 0;
    var z = x;
    for (int i = 0; i < 20; i++) {
      z -= (z * z - x) / (2 * z);
    }
    return z;
  }

  /// Этическая оценка сообщения
  static EthicsResult evaluate(String text) {
    final msgVec = textToVector(text);
    final score = cosineSimilarity(ethicsVector, msgVec);
    final weight = 0.3 + score * 0.5;

    return EthicsResult(
      score: score,
      weight: weight,
      allowed: weight >= minWeight,
    );
  }
}

/// Результат этической проверки
class EthicsResult {
  final double score;
  final double weight;
  final bool allowed;

  EthicsResult({
    required this.score,
    required this.weight,
    required this.allowed,
  });
}