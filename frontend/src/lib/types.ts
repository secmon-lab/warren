export interface Ticket {
  id: string;
  status: string;
  title: string;
  description: string;
  summary: string;
  assignee?: User;
  alerts: Alert[];
  comments: Comment[];
  conclusion?: string;
  reason?: string;
  finding?: Finding;
  createdAt: string;
  updatedAt: string;
}

export interface Comment {
  id: string;
  content: string;
  createdAt: string;
  updatedAt: string;
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
}

export interface Finding {
  severity: string;
  summary: string;
  reason: string;
  recommendation: string;
}

export type TicketStatus = 'open' | 'pending' | 'resolved' | 'archived';

export const TICKET_STATUS_LABELS = {
  open: '🔍 Open',
  pending: '🕒 Pending',
  resolved: '✅️ Resolved',
  archived: '📦 Archived',
} as const;

export const TICKET_STATUS_COLORS = {
  open: 'bg-blue-100 text-blue-800',
  pending: 'bg-yellow-100 text-yellow-800',
  resolved: 'bg-green-100 text-green-800',
  archived: 'bg-gray-100 text-gray-800',
} as const; 