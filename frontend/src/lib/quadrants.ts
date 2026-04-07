export const QUADRANT_COLORS: Record<string, string> = {
  working: "#22c55e",
  leech: "#ef4444",
  "hidden-gem": "#8b5cf6",
  noise: "#f59e0b",
  "insufficient-data": "#9ca3af",
};

export const QUADRANT_LABELS: Record<string, string> = {
  working: "Working",
  leech: "Leech",
  "hidden-gem": "Hidden Gem",
  noise: "Noise",
  "insufficient-data": "Insufficient Data",
};

export function quadrantLabel(q: string): string {
  return QUADRANT_LABELS[q] ?? q;
}

export function quadrantColor(q: string): string {
  return QUADRANT_COLORS[q] ?? "#6b7280";
}
