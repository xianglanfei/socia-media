import 'dart:convert';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import '../models/user.dart';
import 'api_service.dart';

class AuthService extends ChangeNotifier {
  static const _storage = FlutterSecureStorage();
  User? _currentUser;
  String? _authToken;

  User? get currentUser => _currentUser;
  String? get authToken => _authToken;
  bool get isLoggedIn => _authToken != null;

  User get user {
    if (_currentUser == null) {
      throw Exception('User not logged in');
    }
    return _currentUser!;
  }

  // Initialize service - try to load saved credentials
  Future<void> initialize() async {
    final token = await _storage.read(key: 'auth_token');
    final userJson = await _storage.read(key: 'user_data');

    if (token != null && userJson != null) {
      _authToken = token;
      _currentUser = User.fromJson(jsonDecode(userJson));
      notifyListeners();
    }
  }

  Future<void> sendVerificationCode(String phone) async {
    final apiService = ApiService();
    final response = await apiService.sendVerificationCode(phone);

    if (response.statusCode != 200) {
      final error = jsonDecode(response.body);
      throw Exception(error['message'] ?? '发送验证码失败');
    }
  }

  Future<User> register({
    required String phone,
    required String code,
    required String nickname,
    String? gender,
    int? age,
    String flirtStyle = 'humorous',
  }) async {
    final apiService = ApiService();
    final response = await apiService.register(
      phone: phone,
      code: code,
      nickname: nickname,
      gender: gender,
      age: age,
      flirtStyle: flirtStyle,
    );

    if (response.statusCode == 201) {
      final data = jsonDecode(response.body);
      _currentUser = User.fromJson(data['user']);
      _authToken = data['token'];
      apiService.setAuthToken(_authToken!);
      await _saveCredentials();
      notifyListeners();
      return _currentUser!;
    } else {
      final error = jsonDecode(response.body);
      throw Exception(error['message'] ?? '注册失败');
    }
  }

  Future<User> login({
    required String phone,
    required String code,
  }) async {
    final apiService = ApiService();
    final response = await apiService.login(phone: phone, code: code);

    if (response.statusCode == 200) {
      final data = jsonDecode(response.body);
      _currentUser = User.fromJson(data['user']);
      _authToken = data['token'];
      apiService.setAuthToken(_authToken!);
      await _saveCredentials();
      notifyListeners();
      return _currentUser!;
    } else {
      final error = jsonDecode(response.body);
      throw Exception(error['message'] ?? '登录失败');
    }
  }

  Future<void> logout() async {
    try {
      final apiService = ApiService();
      await apiService.logout();
    } catch (e) {
      // Continue with local logout even if API call fails
    } finally {
      await _clearCredentials();
      _currentUser = null;
      _authToken = null;
      notifyListeners();
    }
  }

  Future<void> updateFlirtStyle(String flirtStyle) async {
    final apiService = ApiService();
    final updatedUser = await apiService.updateFlirtStyle(flirtStyle);
    _currentUser = updatedUser;
    await _saveCredentials();
    notifyListeners();
  }

  Future<void> updateProfile({
    String? nickname,
    String? gender,
    int? age,
    String? bio,
    String? avatarUrl,
  }) async {
    final apiService = ApiService();
    final updatedUser = await apiService.updateMyProfile(
      nickname: nickname,
      gender: gender,
      age: age,
      bio: bio,
      avatarUrl: avatarUrl,
    );
    _currentUser = updatedUser;
    await _saveCredentials();
    notifyListeners();
  }

  Future<void> _saveCredentials() async {
    if (_authToken != null) {
      await _storage.write(key: 'auth_token', value: _authToken);
    }
    if (_currentUser != null) {
      await _storage.write(
        key: 'user_data',
        value: jsonEncode(_currentUser!.toJson()),
      );
    }
  }

  Future<void> _clearCredentials() async {
    await _storage.delete(key: 'auth_token');
    await _storage.delete(key: 'user_data');
  }
}
