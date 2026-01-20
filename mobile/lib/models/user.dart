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

  // Flirt style display names in Chinese
  static const Map<String, String> flirtStyleNames = {
    'direct': '直球型',
    'humorous': '幽默风趣',
    'romantic': '温柔浪漫',
    'subtle': '含蓄内敛',
  };

  String get flirtStyleName => flirtStyleNames[flirtStyle] ?? flirtStyle;

  // Flirt style descriptions
  static const Map<String, String> flirtStyleDescriptions = {
    'direct': '直接、自信 - 适合喜欢直来直去的人',
    'humorous': '机智、有趣 - 适合喜欢轻松愉快氛围的人',
    'romantic': '温暖、浪漫 - 适合喜欢浪漫氛围的人',
    'subtle': '含蓄、深情 - 适合不张扬但有意义的人',
  };

  String get flirtStyleDescription => flirtStyleDescriptions[flirtStyle] ?? '';
}
