class Conversation {
  final String id;
  final String user1Id;
  final String user2Id;
  final DateTime lastMessageAt;
  final User? otherUser;
  final Message? lastMessage;
  final int unreadCount;
  final int stage;

  Conversation({
    required this.id,
    required this.user1Id,
    required this.user2Id,
    required this.lastMessageAt,
    this.otherUser,
    this.lastMessage,
    this.unreadCount = 0,
    this.stage = 0,
  });

  factory Conversation.fromJson(Map<String, dynamic> json) {
    return Conversation(
      id: json['id'] as String,
      user1Id: json['user1_id'] as String,
      user2Id: json['user2_id'] as String,
      lastMessageAt: DateTime.parse(json['last_message_at'] as String),
      otherUser: json['other_user'] != null
          ? User.fromJson(json['other_user'] as Map<String, dynamic>)
          : null,
      lastMessage: json['last_message'] != null
          ? Message.fromJson(json['last_message'] as Map<String, dynamic>)
          : null,
      unreadCount: json['unread_count'] as int? ?? 0,
      stage: json['stage'] as int? ?? 0,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'user1_id': user1Id,
      'user2_id': user2Id,
      'last_message_at': lastMessageAt.toIso8601String(),
      'other_user': otherUser?.toJson(),
      'last_message': lastMessage?.toJson(),
      'unread_count': unreadCount,
      'stage': stage,
    };
  }

  String get stageName => Message.stageNames[stage] ?? '未知';

  bool get isColdStart => stage == 0;
  bool get isBreakingIce => stage == 1;
  bool get isWarmUp => stage == 2;
  bool get isFlirty => stage == 3;
  bool get isDeep => stage == 4;
}

// Forward declarations for imported classes
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

class Message {
  final String id;
  final String conversationId;
  final String senderId;
  final String content;
  final String messageType;
  final String status;
  final DateTime createdAt;

  Message({
    required this.id,
    required this.conversationId,
    required this.senderId,
    required this.content,
    required this.messageType,
    required this.status,
    required this.createdAt,
  });

  factory Message.fromJson(Map<String, dynamic> json) {
    return Message(
      id: json['id'] as String,
      conversationId: json['conversation_id'] as String,
      senderId: json['sender_id'] as String,
      content: json['content'] as String,
      messageType: json['message_type'] as String? ?? 'text',
      status: json['status'] as String? ?? 'sent',
      createdAt: DateTime.parse(json['created_at'] as String),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'conversation_id': conversationId,
      'sender_id': senderId,
      'content': content,
      'message_type': messageType,
      'status': status,
      'created_at': createdAt.toIso8601String(),
    };
  }

  bool get isSent => status == 'sent';
  bool get isDelivered => status == 'delivered';
  bool get isRead => status == 'read';

  bool get isText => messageType == 'text';
  bool get isImage => messageType == 'image';
  bool get isVoice => messageType == 'voice';

  static const Map<int, String> stageNames = {
    0: '冷启动',
    1: '破冰',
    2: '热身',
    3: '暧昧',
    4: '深入',
  };
}
