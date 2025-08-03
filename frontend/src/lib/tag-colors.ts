// Chip colors that match the backend color palette
const chipColors = [
  "bg-red-100 text-red-800",
  "bg-orange-100 text-orange-800", 
  "bg-amber-100 text-amber-800",
  "bg-yellow-100 text-yellow-800",
  "bg-lime-100 text-lime-800",
  "bg-green-100 text-green-800",
  "bg-emerald-100 text-emerald-800",
  "bg-teal-100 text-teal-800",
  "bg-cyan-100 text-cyan-800",
  "bg-sky-100 text-sky-800",
  "bg-blue-100 text-blue-800",
  "bg-indigo-100 text-indigo-800",
  "bg-violet-100 text-violet-800",
  "bg-purple-100 text-purple-800",
  "bg-fuchsia-100 text-fuchsia-800",
  "bg-pink-100 text-pink-800",
  "bg-rose-100 text-rose-800",
  "bg-slate-100 text-slate-800",
  "bg-gray-100 text-gray-800",
  "bg-zinc-100 text-zinc-800",
];

// Simple hash function to generate deterministic colors for tag names
function simpleHash(str: string): number {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // Convert to 32-bit integer
  }
  return Math.abs(hash);
}

// Generate a deterministic color for a tag name
// This should match the backend color generation logic
export function generateTagColor(tagName: string): string {
  const hash = simpleHash(tagName);
  const colorIndex = hash % chipColors.length;
  return chipColors[colorIndex];
}