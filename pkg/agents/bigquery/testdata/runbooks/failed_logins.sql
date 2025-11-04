-- Title: Failed Login Investigation
-- Description: Query to find failed login attempts by IP address and user
SELECT
  timestamp,
  user_email,
  source_ip,
  error_code,
  user_agent
FROM `project.dataset.auth_logs`
WHERE
  event_type = 'login_failed'
  AND timestamp >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)
ORDER BY timestamp DESC
LIMIT 100
