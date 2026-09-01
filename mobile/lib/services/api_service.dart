import 'dart:async';
import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/message.dart';

class ApiService {
  final String baseUrl;

  ApiService({required this.baseUrl});

  Future<List<Message>> getMessages({int limit = 200, String? channel}) async {
    try {
      final params = <String, String>{'limit': limit.toString()};
      if (channel != null) params['channel'] = channel;

      final uri = Uri.parse('$baseUrl/messages').replace(queryParameters: params);
      final response = await http.get(uri).timeout(const Duration(seconds: 5));

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        if (data is List) {
          return data
              .whereType<Map<String, dynamic>>()
              .map((m) => Message.fromJson(m))
              .toList();
        } else if (data is Map && data['messages'] != null) {
          final List messages = data['messages'];
          return messages
              .whereType<Map<String, dynamic>>()
              .map((m) => Message.fromJson(m))
              .toList();
        }
      }
    } catch (_) {}
    return [];
  }

  Future<Map<String, dynamic>?> sendMessage(
    String text, {
    String? channel,
    int ttl = 0,
    String? id,
  }) async {
    try {
      final uri = Uri.parse('$baseUrl/send');
      final body = <String, dynamic>{
        'message': text,
        'ttl': ttl,
        if (id != null) 'id': id,
      };
      if (channel != null) body['channel'] = channel;

      final response = await http.post(
        uri,
        headers: {'Content-Type': 'application/json'},
        body: json.encode(body),
      ).timeout(const Duration(seconds: 5));

      if (response.statusCode == 200) {
        return json.decode(response.body);
      }
    } catch (_) {}
    return null;
  }

  Future<Map<String, dynamic>?> getStatus() async {
    try {
      final uri = Uri.parse('$baseUrl/status');
      final response = await http.get(uri).timeout(const Duration(seconds: 5));

      if (response.statusCode == 200) {
        return json.decode(response.body);
      }
    } catch (_) {}
    return null;
  }

  Future<List<Map<String, dynamic>>> getChannels() async {
    try {
      final uri = Uri.parse('$baseUrl/channels');
      final response = await http.get(uri).timeout(const Duration(seconds: 5));

      if (response.statusCode == 200) {
        final data = json.decode(response.body);
        if (data is List) {
          return data.whereType<Map<String, dynamic>>().toList();
        } else if (data is Map && data['channels'] != null) {
          final List channels = data['channels'];
          return channels.whereType<Map<String, dynamic>>().toList();
        }
      }
    } catch (_) {}
    return [];
  }

  Future<void> sendFeedback(String id, int score) async {
    try {
      final uri = Uri.parse('$baseUrl/feedback?id=$id&score=$score');
      await http.post(uri).timeout(const Duration(seconds: 5));
    } catch (_) {}
  }
}