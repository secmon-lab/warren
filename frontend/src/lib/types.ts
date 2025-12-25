export interface Ticket {
  id: string;
  status: string;
  title: string;
  description: string;
  summary: string;
  assignee?: User;
  alerts?: Alert[]; // Optional because it's not always fetched from GraphQL
  comments: Comment[];
  alertsCount: number;
  commentsCount: number;
  conclusion?: string;
  reason?: string;
  finding?: Finding;
  slackLink?: string;
  createdAt: string;
  updatedAt: string;
  isTest: boolean;
  tags?: string[];
  tagObjects?: { id: string; name: string }[];
}

export interface Comment {
  id: string;
  content: string;
  user?: User;
  createdAt: string;
  updatedAt: string;
}

export interface CommentsResponse {
  comments: Comment[];
  totalCount: number;
}

export interface Alert {
  id: string;
  title: string;
  description?: string;
  schema: string;
  data: string;
  attributes: AlertAttribute[];
  createdAt: string;
  ticket?: Ticket;
  tags?: string[];
  tagObjects?: { id: string; name: string }[];
}

export interface AlertAttribute {
  key: string;
  value: string;
  link?: string;
  auto: boolean;
}

export interface User {
  id: string;
  name: string;
  icon?: string;
}

export interface Finding {
  severity: string;
  summary: string;
  reason: string;
  recommendation: string;
}

export type TicketStatus = "open" | "pending" | "resolved" | "archived";

export const TICKET_STATUS_LABELS = {
  open: "🔍 Open",
  pending: "🕒 Pending",
  resolved: "✅️ Resolved",
  archived: "📦 Archived",
} as const;

export const TICKET_STATUS_COLORS = {
  open: "bg-blue-100 text-blue-800",
  pending: "bg-yellow-100 text-yellow-800",
  resolved: "bg-green-100 text-green-800",
  archived: "bg-gray-100 text-gray-800",
} as const;

// Alert conclusions
export type AlertConclusion =
  | "intended"
  | "unaffected"
  | "false_positive"
  | "true_positive"
  | "escalated";

export const ALERT_CONCLUSION_LABELS: Record<AlertConclusion, string> = {
  intended: "👍 Intended",
  unaffected: "🛡️ Unaffected",
  false_positive: "🚫 False Positive",
  true_positive: "🚨 True Positive",
  escalated: "⬆️ Escalated",
};

export const ALERT_CONCLUSION_DESCRIPTIONS: Record<AlertConclusion, string> = {
  intended: "The alert is intended behavior or configuration.",
  unaffected:
    "The alert indicates actual attack or vulnerability, but it is no impact.",
  false_positive: "The alert is not attack or impact on the system.",
  true_positive: "The alert has actual impact on the system.",
  escalated: "The alert has been escalated to external management.",
};

export interface TagMetadata {
  id: string;
  name: string;
  description?: string;
  color: string;
  createdAt: string;
  updatedAt: string;
}

export interface Knowledge {
  slug: string;
  name: string;
  topic: string;
  content: string;
  commitID: string;
  author: User;
  createdAt: string;
  updatedAt: string;
  state: string;
}

export interface TopicSummary {
  topic: string;
  count: number;
}

export interface Session {
  id: string;
  ticketID: string;
  status: string; // 'running' | 'completed' | 'aborted'
  createdAt: string;
  updatedAt: string;
}

export interface SessionMessage {
  id: string;
  sessionID: string;
  type: string; // 'trace' | 'plan' | 'response'
  content: string;
  createdAt: string;
  updatedAt: string;
}
