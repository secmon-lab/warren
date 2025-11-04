-- title: Suspicious Access Pattern
-- description: Identify unusual access patterns from specific IP addresses
SELECT
  timestamp,
  source_ip,
  COUNT(*) as access_count,
  ARRAY_AGG(DISTINCT user_email) as users
FROM `project.dataset.access_logs`
WHERE
  timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)
GROUP BY timestamp, source_ip
HAVING access_count > 10
ORDER BY access_count DESC
LIMIT 50
