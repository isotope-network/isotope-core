import 'dart:convert';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../models/message.dart';

class WsService {
  final String wsUrl;
  WebSocketChannel? _channel;
  bool _connected = false;

  void Function(Message message)? onMessage;
  void Function(String msgId, String status, String channel)? onStatus;
  void Function(String channel)? onSubscribed;
  void Function()? onDisconnected;

  WsService({required this.wsUrl});

  bool get isConnected => _connected;

  void connect() {
    try {
      _channel = WebSocketChannel.connect(Uri.parse(wsUrl));
      _connected = true;

      _channel!.stream.listen(
        (data) {
          _handleMessage(data);
        },
        onDone: () {
          _connected = false;
          onDisconnected?.call();
        },
        onError: (_) {
          _connected = false;
          onDisconnected?.call();
        },
      );
    } catch (_) {
      _connected = false;
    }
  }

  void disconnect() {
    _channel?.sink.close();
    _connected = false;
  }

  void sendMessage(String text, {String channel = 'general'}) {
    if (!_connected) return;
    _send({
      'type': 'send',
      'channel': channel,
      'message': text,
    });
  }

  void subscribe(String channel) {
    if (!_connected) return;
    _send({
      'type': 'subscribe',
      'channel': channel,
    });
  }

  void sendFeedback(String id, int score) {
    if (!_connected) return;
    _send({
      'type': 'feedback',
      'id': id,
      'score': score,
    });
  }

  void _send(Map<String, dynamic> data) {
    _channel?.sink.add(json.encode(data));
  }

  void _handleMessage(dynamic data) {
    try {
      final json = jsonDecode(data);
      final type = json['type'] as String?;
      print('WS received: $type');

      switch (type) {
        case 'message':
          final msg = Message.fromJson(json['data']);
          onMessage?.call(msg);
          break;

        case 'status':
          onStatus?.call(
            json['msgID'] ?? '',
            json['deliveryStatus'] ?? '',
            json['channel'] ?? '',
          );
          break;

        case 'subscribed':
          onSubscribed?.call(json['channel'] ?? '');
          break;
      }
    } catch (_) {}
  }
}