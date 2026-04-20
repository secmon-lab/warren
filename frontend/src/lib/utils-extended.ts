import { formatDistanceToNow, format } from 'date-fns';
import { enUS } from 'date-fns/locale';

export function formatRelativeTime(dateString: string): string {
  if (!dateString) return "";
  const date = new Date(dateString);
  if (isNaN(date.getTime()) || date.getFullYear() <= 1970) {
    // Guard against zero / empty / invalid timestamps that would
    // otherwise render as "over 2025 years ago".
    return "";
  }
  return formatDistanceToNow(date, {
    addSuffix: true,
    locale: enUS,
  });
}

export function formatDateTime(dateString: string): string {
  const date = new Date(dateString);
  return format(date, 'MMM dd, yyyy HH:mm', { locale: enUS });
}

export function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return format(date, 'MMM dd, yyyy', { locale: enUS });
}

export function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return text.substring(0, maxLength) + '...';
}

export function generateSlackLink(ticketId: string): string {
  // TODO: Replace with actual Slack workspace URL
  return `https://your-workspace.slack.com/archives/CHANNEL_ID/${ticketId}`;
}

export function generateAlertSlackLink(alertId: string): string {
  // TODO: Replace with actual Slack workspace URL
  return `https://your-workspace.slack.com/archives/CHANNEL_ID/${alertId}`;
} 