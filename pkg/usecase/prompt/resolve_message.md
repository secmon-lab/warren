# Thoughtful Conclusion Message Generation for Ticket Resolution

Generate a warm and thoughtful conclusion message that acknowledges the responder's work and provides a satisfying closure to the ticket resolution process, taking into account their specific efforts and approach.

## Requirements
- About 1 line of short message
- Provides thoughtful acknowledgment and satisfying closure
- **Reflects the actual response process based on the conversation history**
- **Acknowledges specific challenges, teamwork, or insights from the comments**
- Creates a supportive and positive atmosphere
- Output in {{ .lang }}
- Use 1-2 emojis

## Example Tone (adapt based on conversation context)
**Basic appreciation**:
- "Another peaceful resolution thanks to your careful attention. Time for a well-deserved coffee break ☕✨"
- "Excellent work! This ticket will be remembered as a textbook example 📚😌"
- "The servers are breathing a sigh of relief. Thank you for keeping things stable 🖥️💤"

**Context-aware examples** (adjust based on conversation history):
- For late-night responses: "Late night heroics again! Tomorrow's coffee will taste extra special ☕🌅"
- For team collaboration: "Beautiful teamwork! The coordination was impressive to watch 🤝✨"
- For thorough investigation: "Your persistent investigation paid off. Great detective work! 🔍💎"
- For quick resolution: "Lightning-fast response! That speed is becoming legendary ⚡"
- For complex cases: "Masterful handling of a tricky situation. This experience is gold 💎"
- For learning moments: "Great problem-solving with bonus learning! Knowledge gained 📚✨"

## Ticket Information
- Title: {{ .title }}
- Conclusion: {{ .conclusion }}
- Reason: {{ .reason }}

## Comments

{{ range .comments }}
- {{ .User.Name }}: {{ .Comment }}
{{ end }}

Based on the above ticket information and conversation history, please generate one thoughtful conclusion message that provides satisfying closure while acknowledging the responder's specific efforts and approach.

**Key instructions**:
- **Analyze the conversation history to understand**: timing (late night?), collaboration (multiple people?), complexity (investigation needed?), speed of resolution, any learning or discoveries
- **Tailor the message** to acknowledge the specific context and approach taken
- The message should be one line and within 100 characters including emojis
- Focus on providing closure rather than generic congratulations
- Make it feel personal and contextually relevant

Do not include acknowledgment phrases like "OK", "understood", or similar confirmation responses. Output only the thoughtful conclusion message that reflects their actual work process.
