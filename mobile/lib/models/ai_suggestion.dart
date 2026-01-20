class AISuggestion {
  final String conversationId;
  final int stage;
  final List<Suggestion> suggestions;

  AISuggestion({
    required this.conversationId,
    required this.stage,
    required this.suggestions,
  });

  factory AISuggestion.fromJson(Map<String, dynamic> json) {
    final suggestionsList = json['suggestions'] as List;
    return AISuggestion(
      conversationId: json['conversation_id'] as String,
      stage: json['stage'] as int,
      suggestions: suggestionsList
          .map((e) => Suggestion.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'conversation_id': conversationId,
      'stage': stage,
      'suggestions': suggestions.map((e) => e.toJson()).toList(),
    };
  }

  String get stageName => Message.stageNames[stage] ?? '未知';
}

class Suggestion {
  final String text;
  final String style;
  final String reason;

  Suggestion({
    required this.text,
    required this.style,
    required this.reason,
  });

  factory Suggestion.fromJson(Map<String, dynamic> json) {
    return Suggestion(
      text: json['text'] as String,
      style: json['style'] as String,
      reason: json['reason'] as String,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'text': text,
      'style': style,
      'reason': reason,
    };
  }

  // Style display names in Chinese
  static const Map<String, String> styleNames = {
    'direct': '直球型',
    'humorous': '幽默风趣',
    'romantic': '温柔浪漫',
    'subtle': '含蓄内敛',
  };

  String get styleName => styleNames[style] ?? style;
}

// Forward declaration for Message
class Message {
  static const Map<int, String> stageNames = {
    0: '冷启动',
    1: '破冰',
    2: '热身',
    3: '暧昧',
    4: '深入',
  };
}
