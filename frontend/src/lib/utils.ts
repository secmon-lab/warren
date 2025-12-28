import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatTimeAgo(dateString: string): string {
  const date = new Date(dateString);
  const now = new Date();
  const diffInHours = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60));

  if (diffInHours < 1) return "Just now";
  if (diffInHours < 24) return `${diffInHours}h ago`;
  const diffInDays = Math.floor(diffInHours / 24);
  if (diffInDays < 7) return `${diffInDays}d ago`;
  // Format as YYYY/MM/DD
  return date.toISOString().split('T')[0].replace(/-/g, '/');
}

export function getSeverityColor(schema: string): string {
  // Simple heuristic based on schema name
  const lowerSchema = schema.toLowerCase();
  if (lowerSchema.includes('critical') || lowerSchema.includes('high')) {
    return 'bg-red-100 text-red-800 border-red-200';
  }
  if (lowerSchema.includes('medium') || lowerSchema.includes('warning')) {
    return 'bg-yellow-100 text-yellow-800 border-yellow-200';
  }
  if (lowerSchema.includes('low') || lowerSchema.includes('info')) {
    return 'bg-blue-100 text-blue-800 border-blue-200';
  }
  return 'bg-gray-100 text-gray-800 border-gray-200';
}
