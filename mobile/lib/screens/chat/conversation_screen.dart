import 'dart:async';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../services/websocket_service.dart';
import '../../services/api_service.dart';
import '../../models/conversation.dart';
import '../../models/message.dart';
import '../../models/ai_suggestion.dart';

class ConversationScreen extends StatefulWidget {
  final String conversationId;
  final User? otherUser;
  final WebSocketService wsService;

  const ConversationScreen({
    super.key,
    required this.conversationId,
    required this.otherUser,
    required this.wsService,
  });

  @override
  State<ConversationScreen> createState() => _ConversationScreenState();
}

class _ConversationScreenState extends State<ConversationScreen> {
  final TextEditingController _messageController = TextEditingController();
  final ScrollController _scrollController = ScrollController();
  final List<Message> _messages = [];

  bool _isLoading = true;
  bool _isTyping = false;
  bool _showSuggestions = false;
  AISuggestion? _aiSuggestions;
  Timer? _typingTimer;
  Timer? _debounceTimer;

  @override
  void initState() {
    super.initState();
    _loadMessages();
    _listenForMessages();
  }

  Future<void> _loadMessages() async {
    setState(() => _isLoading = true);
    try {
      final apiService = context.read<ApiService>();
      final messages = await apiService.getMessages(widget.conversationId);
      setState(() {
        _messages.clear();
        _messages.addAll(messages);
        _isLoading = false;
      });
      _scrollToBottom();
      await _loadAISuggestions();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString())),
        );
      }
      setState(() => _isLoading = false);
    }
  }

  Future<void> _loadAISuggestions() async {
    try {
      final apiService = context.read<ApiService>();
      final suggestions = await apiService.getAISuggestions(widget.conversationId);
      setState(() {
        _aiSuggestions = suggestions;
        _showSuggestions = true;
      });
    } catch (e) {
      debugPrint('Failed to load AI suggestions: $e');
    }
  }

  void _listenForMessages() {
    widget.wsService.messageStream.listen((message) {
      if (message.conversationId == widget.conversationId) {
        setState(() {
          _messages.add(message);
        });
        _scrollToBottom();
        _refreshAISuggestionsDebounced();
      }
    });

    widget.wsService.eventStream.listen((event) {
      if (event['type'] == 'typing' &&
          event['conversation_id'] == widget.conversationId) {
        final isTyping = event['is_typing'] as bool? ?? false;
        setState(() => _isTyping = isTyping);
      }
    });
  }

  void _refreshAISuggestionsDebounced() {
    _debounceTimer?.cancel();
    _debounceTimer = Timer(const Duration(milliseconds: 500), () {
      _loadAISuggestions();
    });
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollController.hasClients) {
        _scrollController.animateTo(
          _scrollController.position.maxScrollExtent,
          duration: const Duration(milliseconds: 300),
          curve: Curves.easeOut,
        );
      }
    });
  }

  void _handleMessageChanged() {
    if (_messageController.text.isNotEmpty) {
      _sendTyping(true);
      _typingTimer?.cancel();
      _typingTimer = Timer(const Duration(seconds: 2), () {
        _sendTyping(false);
      });
    }
  }

  void _sendTyping(bool isTyping) {
    widget.wsService.sendTyping(widget.conversationId, isTyping);
  }

  Future<void> _sendMessage() async {
    final content = _messageController.text.trim();
    if (content.isEmpty) return;

    _messageController.clear();
    _sendTyping(false);

    // Optimistically add message
    setState(() {
      _showSuggestions = false;
    });

    // Send via WebSocket
    widget.wsService.sendMessage(
      conversationId: widget.conversationId,
      content: content,
    );

    // Also send via API for persistence
    try {
      final apiService = context.read<ApiService>();
      await apiService.sendMessage(widget.conversationId, content);
    } catch (e) {
      debugPrint('Failed to send message via API: $e');
    }
  }

  void _useSuggestion(Suggestion suggestion) {
    _messageController.text = suggestion.text;
    setState(() {
      _showSuggestions = false;
    });
    _sendMessage();
  }

  @override
  void dispose() {
    _messageController.dispose();
    _scrollController.dispose();
    _typingTimer?.cancel();
    _debounceTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final authService = context.read<AuthService>();
    final otherUser = widget.otherUser;

    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              otherUser?.nickname ?? '未知用户',
              style: const TextStyle(fontSize: 18),
            ),
            if (_isTyping)
              const Text(
                '正在输入...',
                style: TextStyle(
                  fontSize: 12,
                  fontStyle: FontStyle.italic,
                ),
              ),
          ],
        ),
        backgroundColor: theme.colorScheme.primary,
        foregroundColor: theme.colorScheme.onPrimary,
      ),
      body: Column(
        children: [
          // Messages list
          Expanded(
            child: _isLoading
                ? const Center(child: CircularProgressIndicator())
                : _buildMessagesList(),
          ),
          // AI Suggestions panel
          if (_showSuggestions && _aiSuggestions != null)
            _buildAISuggestions(),
          // Input area
          _buildInputArea(theme),
        ],
      ),
    );
  }

  Widget _buildMessagesList() {
    if (_messages.isEmpty) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(
              Icons.chat_bubble_outline,
              size: 80,
              color: Colors.grey[400],
            ),
            const SizedBox(height: 16),
            Text(
              '开始聊天吧',
              style: TextStyle(
                fontSize: 18,
                color: Colors.grey[600],
              ),
            ),
          ],
        ),
      );
    }

    final authService = context.read<AuthService>();
    final currentUserId = authService.user.id;

    return ListView.builder(
      controller: _scrollController,
      padding: const EdgeInsets.all(8),
      itemCount: _messages.length,
      itemBuilder: (context, index) {
        final message = _messages[index];
        final isOwn = message.senderId == currentUserId;
        return _buildMessageBubble(message, isOwn);
      },
    );
  }

  Widget _buildMessageBubble(Message message, bool isOwn) {
    return Align(
      alignment: isOwn ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.symmetric(vertical: 4, horizontal: 8),
        constraints: BoxConstraints(
          maxWidth: MediaQuery.of(context).size.width * 0.75,
        ),
        decoration: BoxDecoration(
          color: isOwn
              ? Theme.of(context).colorScheme.primary
              : Colors.grey[300],
          borderRadius: BorderRadius.circular(12).copyWith(
            bottomLeft: isOwn ? const Radius.circular(12) : const Radius.circular(0),
            bottomRight: isOwn ? const Radius.circular(0) : const Radius.circular(12),
          ),
        ),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              message.content,
              style: TextStyle(
                color: isOwn
                    ? Theme.of(context).colorScheme.onPrimary
                    : Colors.black87,
              ),
            ),
            const SizedBox(height: 4),
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  _formatMessageTime(message.createdAt),
                  style: TextStyle(
                    fontSize: 10,
                    color: isOwn
                        ? Theme.of(context).colorScheme.onPrimary.withOpacity(0.7)
                        : Colors.black54,
                  ),
                ),
                if (isOwn)
                  Padding(
                    padding: const EdgeInsets.only(left: 4),
                    child: Icon(
                      _getMessageStatusIcon(message.status),
                      size: 12,
                      color: Theme.of(context)
                          .colorScheme
                          .onPrimary
                          .withOpacity(0.7),
                    ),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  IconData _getMessageStatusIcon(String status) {
    switch (status) {
      case 'delivered':
        return Icons.done_all;
      case 'read':
        return Icons.done_all;
      default:
        return Icons.done;
    }
  }

  String _formatMessageTime(DateTime dateTime) {
    final now = DateTime.now();
    final difference = now.difference(dateTime);

    if (difference.inMinutes < 1) {
      return '刚刚';
    } else if (difference.inHours < 1) {
      return '${difference.inMinutes}分钟前';
    } else if (dateTime.day == now.day) {
      return '${dateTime.hour.toString().padLeft(2, '0')}:${dateTime.minute.toString().padLeft(2, '0')}';
    } else {
      return '${dateTime.month}/${dateTime.day} ${dateTime.hour.toString().padLeft(2, '0')}:${dateTime.minute.toString().padLeft(2, '0')}';
    }
  }

  Widget _buildAISuggestions() {
    final suggestions = _aiSuggestions!.suggestions;

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.amber[50],
        border: Border(
          top: BorderSide(color: Colors.amber[200]!),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                Icons.lightbulb,
                color: Colors.amber[700],
                size: 20,
              ),
              const SizedBox(width: 8),
              Text(
                'AI 建议',
                style: TextStyle(
                  fontWeight: FontWeight.bold,
                  color: Colors.amber[900],
                ),
              ),
              const SizedBox(width: 8),
              Text(
                '(${_aiSuggestions!.stageName})',
                style: TextStyle(
                  fontSize: 12,
                  color: Colors.amber[700],
                ),
              ),
              const Spacer(),
              IconButton(
                icon: const Icon(Icons.close),
                onPressed: () {
                  setState(() => _showSuggestions = false);
                },
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(),
              ),
            ],
          ),
          const SizedBox(height: 8),
          ...List.generate(suggestions.length, (index) {
            final suggestion = suggestions[index];
            return Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: GestureDetector(
                onTap: () => _useSuggestion(suggestion),
                child: Container(
                  padding: const EdgeInsets.all(10),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.amber[300]!),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 6,
                              vertical: 2,
                            ),
                            decoration: BoxDecoration(
                              color: _getStyleColor(suggestion.style),
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: Text(
                              suggestion.styleName,
                              style: const TextStyle(
                                fontSize: 10,
                                color: Colors.white,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 6),
                      Text(
                        suggestion.text,
                        style: const TextStyle(fontSize: 14),
                      ),
                      Text(
                        suggestion.reason,
                        style: TextStyle(
                          fontSize: 11,
                          color: Colors.grey[600],
                          fontStyle: FontStyle.italic,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            );
          }),
        ],
      ),
    );
  }

  Color _getStyleColor(String style) {
    switch (style) {
      case 'direct':
        return Colors.blue;
      case 'humorous':
        return Colors.orange;
      case 'romantic':
        return Colors.pink;
      case 'subtle':
        return Colors.purple;
      default:
        return Colors.grey;
    }
  }

  Widget _buildInputArea(ThemeData theme) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(
          top: BorderSide(color: Colors.grey[300]!),
        ),
      ),
      child: Row(
        children: [
          IconButton(
            icon: const Icon(Icons.emoji_emotions_outlined),
            onPressed: () {
              // TODO: Open emoji picker
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('表情功能即将推出')),
              );
            },
          ),
          IconButton(
            icon: const Icon(Icons.image_outlined),
            onPressed: () {
              // TODO: Open image picker
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(content: Text('图片功能即将推出')),
              );
            },
          ),
          Expanded(
            child: TextField(
              controller: _messageController,
              decoration: const InputDecoration(
                hintText: '点击此处输入自定义回复...',
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.all(Radius.circular(24)),
                ),
                contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 12),
              ),
              maxLines: null,
              textInputAction: TextInputAction.send,
              onChanged: (_) => _handleMessageChanged(),
              onSubmitted: (_) => _sendMessage(),
            ),
          ),
          const SizedBox(width: 8),
          FilledButton.tonal(
            onPressed: _messageController.text.trim().isEmpty
                ? null
                : _sendMessage,
            style: FilledButton.styleFrom(
              shape: const CircleBorder(),
              padding: const EdgeInsets.all(12),
            ),
            child: const Icon(Icons.send),
          ),
        ],
      ),
    );
  }
}

// Forward declaration for User class
class User {
  final String id;
  final String phone;
  final String nickname;
  final String? gender;
  final int? age;
  final String? avatarUrl;
  final String? bio;
  final String flirtStyle;
  final DateTime createdAt;
  final DateTime updatedAt;

  User({
    required this.id,
    required this.phone,
    required this.nickname,
    this.gender,
    this.age,
    this.avatarUrl,
    this.bio,
    required this.flirtStyle,
    required this.createdAt,
    required this.updatedAt,
  });

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      id: json['id'] as String,
      phone: json['phone'] as String,
      nickname: json['nickname'] as String,
      gender: json['gender'] as String?,
      age: json['age'] as int?,
      avatarUrl: json['avatar_url'] as String?,
      bio: json['bio'] as String?,
      flirtStyle: json['flirt_style'] as String? ?? 'humorous',
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'phone': phone,
      'nickname': nickname,
      'gender': gender,
      'age': age,
      'avatar_url': avatarUrl,
      'bio': bio,
      'flirt_style': flirtStyle,
      'created_at': createdAt.toIso8601String(),
      'updated_at': updatedAt.toIso8601String(),
    };
  }
}
