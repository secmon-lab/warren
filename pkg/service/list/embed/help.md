*📋 Basic Usage*
━━━━━━━━━━━━━━━━━━━━━━
• `list from 10:00 to 11:00` _Show alerts within time range_
• `list between 2025-01-01 2025-01-02`
• `list unresolved` _Show unresolved alerts_
• `list after 2025-01-01` _Show alerts after date_
• `list since 10m` _Show alerts from last 10 minutes_
  _(Default: `list since 1d`)_

*🔍 Filters and Modifiers*
━━━━━━━━━━━━━━━━━━━━━━
• `list | grep test` _Filter by keyword_
• `list | sort CreatedAt` _Sort by creation time_
• `list | limit 10` _Limit number of results_
• `list | offset 10` _Skip first N results_
• `list | cluster` _Group similar alerts_
