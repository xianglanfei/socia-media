import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/user.dart';
import '../models/conversation.dart';
import '../models/message.dart';
import '../models/ai_suggestion.dart';

class ApiService {
  static const String baseUrl = 'http://localhost:8080'; // Update with your backend URL
  String? _authToken;

  String get authToken => _authToken ?? '';

  void setAuthToken(String token) {
    _authToken = token;
  }

  void clearAuthToken() {
    _authToken = null;
  }

  Map<String, String> get headers => {
        'Content-Type': 'application/json',
        if (_authToken != null) 'Authorization': 'Bearer $_authToken',
      };

  // Auth APIs
  Future<http.Response> sendVerificationCode(String phone) async {
    return await http.post(
      Uri.parse('$baseUrl/api/auth/send-code'),
      headers: headers,
      body: jsonEncode({'phone': phone}),
    );
  }

  Future<http.Response> register({
    required String phone,
    required String code,
    required String nickname,
    String? gender,
    int? age,
    String flirtStyle = 'humorous',
  }) async {
    return await http.post(
      Uri.parse('$baseUrl/api/auth/register'),
      headers: headers,
      body: jsonEncode({
        'phone': phone,
        'code': code,
        'nickname': nickname,
        'gender': gender,
        'age': age,
        'flirt_style': flirtStyle,
      }),
    );
  }

  Future<http.Response> login({
    required String phone,
    required String code,
  }) async {
    return await http.post(
      Uri.parse('$baseUrl/api/auth/login'),
      headers: headers,
      body: jsonEncode({'phone': phone, 'code': code}),
    );
  }

  Future<http.Response> logout() async {
    return await http.post(
      Uri.parse('$baseUrl/api/auth/logout'),
      headers: headers,
    );
  }

  // Conversation APIs
  Future<List<Conversation>> getConversations() async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/conversations'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      final List<dynamic> data = jsonDecode(response.body)['conversations'];
      return data.map((e) => Conversation.fromJson(e)).toList();
    } else {
      throw Exception('Failed to load conversations');
    }
  }

  Future<List<Message>> getMessages(String conversationId) async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/conversations/$conversationId/messages'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      final List<dynamic> data = jsonDecode(response.body)['messages'];
      return data.map((e) => Message.fromJson(e)).toList();
    } else {
      throw Exception('Failed to load messages');
    }
  }

  Future<Message> sendMessage(String conversationId, String content) async {
    final response = await http.post(
      Uri.parse('$baseUrl/api/conversations/$conversationId/messages'),
      headers: headers,
      body: jsonEncode({'content': content}),
    );

    if (response.statusCode == 201) {
      return Message.fromJson(jsonDecode(response.body));
    } else {
      throw Exception('Failed to send message');
    }
  }

  // Profile APIs
  Future<User> getMyProfile() async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/profile/me'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      return User.fromJson(jsonDecode(response.body));
    } else {
      throw Exception('Failed to load profile');
    }
  }

  Future<User> updateMyProfile({
    String? nickname,
    String? gender,
    int? age,
    String? bio,
    String? avatarUrl,
  }) async {
    final response = await http.put(
      Uri.parse('$baseUrl/api/profile/me'),
      headers: headers,
      body: jsonEncode({
        if (nickname != null) 'nickname': nickname,
        if (gender != null) 'gender': gender,
        if (age != null) 'age': age,
        if (bio != null) 'bio': bio,
        if (avatarUrl != null) 'avatar_url': avatarUrl,
      }),
    );

    if (response.statusCode == 200) {
      return User.fromJson(jsonDecode(response.body));
    } else {
      throw Exception('Failed to update profile');
    }
  }

  Future<User> updateFlirtStyle(String flirtStyle) async {
    final response = await http.put(
      Uri.parse('$baseUrl/api/profile/flirt-style'),
      headers: headers,
      body: jsonEncode({'flirt_style': flirtStyle}),
    );

    if (response.statusCode == 200) {
      return User.fromJson(jsonDecode(response.body));
    } else {
      throw Exception('Failed to update flirt style');
    }
  }

  Future<User> getOtherProfile(String userId) async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/profile/users/$userId'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      return User.fromJson(jsonDecode(response.body));
    } else {
      throw Exception('Failed to load profile');
    }
  }

  // AI Suggestions API
  Future<AISuggestion> getAISuggestions(String conversationId) async {
    final response = await http.get(
      Uri.parse('$baseUrl/api/ai/suggestions/$conversationId'),
      headers: headers,
    );

    if (response.statusCode == 200) {
      return AISuggestion.fromJson(jsonDecode(response.body));
    } else {
      throw Exception('Failed to load AI suggestions');
    }
  }
}
