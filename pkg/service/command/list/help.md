`list` command shows alerts that are *not bound to any ticket*.

*📋 Basic Usage*
━━━━━━━━━━━━━━━━━━━━━━
• `list from 10:00 to 11:00` _Show alerts within time range_
• `list after 2025-01-01` _Show alerts after date_
• `list since 10m` _Show alerts from last 10 minutes_
• `list all` _Show all unbound alerts_
  _(Default: `list all`)_

*🔍 Filters and Modifiers*
━━━━━━━━━━━━━━━━━━━━━━
• `list | grep test` _Filter by keyword for original alert data_
• `list | sort CreatedAt` _Sort by creation time_
• `list | limit 10` _Limit number of results_
• `list | offset 10` _Skip first N results_
• `list | user @username` _Filter by username_
