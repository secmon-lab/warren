`aggregate` command generates a list of alerts by their alert data and shows the count of each group.

*📋 Basic Usage*
━━━━━━━━━━━━━━━━━━━━━━
• `aggregate` _Group alerts by default fields (Title, Description)_
• `aggregate threshold 0.99` _Show groups with count >= 0.99_
• `aggregate top 10` _Show top 10 groups by count_
• `aggregate th 0.99 top 10` _Show top 10 groups with count >= 0.99_

default threshold is 0.99 and top is 10.
