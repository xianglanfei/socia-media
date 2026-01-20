import 'dart:async';
import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import '../models/message.dart';
import 'api_service.dart';

enum WebSocketStatus {
  connected,
  disconnected,
  connecting,
  error,
}

class WebSocketService extends ChangeNotifier {
  WebSocketChannel? _channel;
  StreamSubscription? _subscription;
  WebSocketStatus _status = WebSocketStatus.disconnected;
  final String _userId;
  final ApiService _apiService;

  WebSocketStatus get status => _status;

  final StreamController<Message> _messageController =
      StreamController<Message>.broadcast();
  final StreamController<Map<String, dynamic>> _eventController =
      StreamController<Map<String, dynamic>>.broadcast();

  Stream<Message> get messageStream => _messageController.stream;
  Stream<Map<String, dynamic>> get eventStream => _eventController.stream;

  WebSocketService(this._userId, this._apiService);

  void connect() {
    if (_status == WebSocketStatus.connected || _status == WebSocketStatus.connecting) {
      return;
    }

    _status = WebSocketStatus.connecting;
    notifyListeners();

    try {
      final wsUrl = Uri.parse('ws://localhost:8080/ws?token=${_apiService.authToken}');
      _channel = WebSocketChannel.connect(wsUrl);

      _subscription = _channel!.stream.listen(
        _handleMessage,
        onError: _handleError,
        onDone: _handleDone,
      );

      // Send connect event
      _send({'type': 'connect', 'user_id': _userId});
    } catch (e) {
      _status = WebSocketStatus.error;
      notifyListeners();
      debugPrint('WebSocket connection error: $e');
    }
  }

  void _handleMessage(dynamic data) {
    _status = WebSocketStatus.connected;
    notifyListeners();

    if (data is String) {
      try {
        final event = jsonDecode(data) as Map<String, dynamic>;
        final type = event['type'] as String?;

        switch (type) {
          case 'message':
            final message = Message.fromJson(event['message'] as Map<String, dynamic>);
            _messageController.add(message);
            break;
          case 'typing':
          case 'read':
          case 'delivered':
            _eventController.add(event);
            break;
          case 'connect':
            debugPrint('WebSocket connected');
            break;
          default:
            _eventController.add(event);
        }
      } catch (e) {
        debugPrint('Error parsing WebSocket message: $e');
      }
    }
  }

  void _handleError(dynamic error) {
    _status = WebSocketStatus.error;
    notifyListeners();
    debugPrint('WebSocket error: $error');
  }

  void _handleDone() {
    _status = WebSocketStatus.disconnected;
    notifyListeners();
    debugPrint('WebSocket connection closed');
  }

  void _send(Map<String, dynamic> data) {
    try {
      _channel?.sink.add(jsonEncode(data));
    } catch (e) {
      debugPrint('Error sending WebSocket message: $e');
    }
  }

  void sendMessage({
    required String conversationId,
    required String content,
    String messageType = 'text',
  }) {
    _send({
      'type': 'message',
      'conversation_id': conversationId,
      'content': content,
      'message_type': messageType,
    });
  }

  void sendTyping(String conversationId, bool isTyping) {
    _send({
      'type': 'typing',
      'conversation_id': conversationId,
      'is_typing': isTyping,
    });
  }

  void markAsRead(String conversationId, List<String> messageIds) {
    _send({
      'type': 'read',
      'conversation_id': conversationId,
      'message_ids': messageIds,
    });
  }

  void disconnect() {
    _send({'type': 'disconnect'});
    _subscription?.cancel();
    _channel?.sink.close();
    _channel = null;
    _status = WebSocketStatus.disconnected;
    notifyListeners();
  }

  @override
  void dispose() {
    disconnect();
    _messageController.close();
    _eventController.close();
    super.dispose();
  }
}
