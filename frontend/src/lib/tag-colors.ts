// Color mappings for tag components
// Single source of truth for color name to Tailwind class mappings
export const TAG_COLOR_MAP: Record<string, string> = {
  red: "bg-red-100 text-red-800",
  orange: "bg-orange-100 text-orange-800",
  amber: "bg-amber-100 text-amber-800",
  yellow: "bg-yellow-100 text-yellow-800",
  lime: "bg-lime-100 text-lime-800",
  green: "bg-green-100 text-green-800",
  emerald: "bg-emerald-100 text-emerald-800",
  teal: "bg-teal-100 text-teal-800",
  cyan: "bg-cyan-100 text-cyan-800",
  sky: "bg-sky-100 text-sky-800",
  blue: "bg-blue-100 text-blue-800",
  indigo: "bg-indigo-100 text-indigo-800",
  violet: "bg-violet-100 text-violet-800",
  purple: "bg-purple-100 text-purple-800",
  fuchsia: "bg-fuchsia-100 text-fuchsia-800",
  pink: "bg-pink-100 text-pink-800",
  rose: "bg-rose-100 text-rose-800",
  slate: "bg-slate-100 text-slate-800",
  gray: "bg-gray-100 text-gray-800",
  zinc: "bg-zinc-100 text-zinc-800",
};

// Reverse mapping: Tailwind class to color name
export const TAG_CLASS_MAP: Record<string, string> = Object.fromEntries(
  Object.entries(TAG_COLOR_MAP).map(([name, className]) => [className, name])
);

// Default fallback color
export const DEFAULT_TAG_COLOR = "gray";
export const DEFAULT_TAG_CLASS = TAG_COLOR_MAP[DEFAULT_TAG_COLOR];

// Array of chip colors for backward compatibility and generateTagColor function
const chipColors = Object.values(TAG_COLOR_MAP);

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

/**
 * Convert a Tailwind color class to a user-friendly color name
 * @param colorClass - Tailwind color class (e.g., "bg-red-100 text-red-800")
 * @returns Color name (e.g., "red") or default fallback
 */
export function colorClassToName(colorClass: string): string {
  return TAG_CLASS_MAP[colorClass] || DEFAULT_TAG_COLOR;
}

/**
 * Convert a user-friendly color name to a Tailwind color class
 * @param colorName - Color name (e.g., "red")
 * @returns Tailwind color class (e.g., "bg-red-100 text-red-800") or default fallback
 */
export function colorNameToClass(colorName: string): string {
  return TAG_COLOR_MAP[colorName] || DEFAULT_TAG_CLASS;
}

/**
 * Get all available color names
 * @returns Array of color names
 */
export function getAvailableColorNames(): string[] {
  return Object.keys(TAG_COLOR_MAP);
}

/**
 * Get all available color classes
 * @returns Array of Tailwind color classes
 */
export function getAvailableColorClasses(): string[] {
  return Object.values(TAG_COLOR_MAP);
}

/**
 * Check if a color name is valid
 * @param colorName - Color name to validate
 * @returns Whether the color name is valid
 */
export function isValidColorName(colorName: string): boolean {
  return colorName in TAG_COLOR_MAP;
}

/**
 * Check if a color class is valid
 * @param colorClass - Color class to validate
 * @returns Whether the color class is valid
 */
export function isValidColorClass(colorClass: string): boolean {
  return colorClass in TAG_CLASS_MAP;
}