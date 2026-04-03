import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";
import type { Memory } from "@/lib/types";

interface EffectivenessHistogramProps {
  memories: Memory[];
}

const BUCKET_SIZE = 10;
const BUCKET_COUNT = 10;

export default function EffectivenessHistogram({
  memories,
}: EffectivenessHistogramProps) {
  const buckets = Array.from({ length: BUCKET_COUNT }, (_, i) => ({
    range: `${i * BUCKET_SIZE}-${(i + 1) * BUCKET_SIZE}%`,
    count: 0,
  }));

  for (const m of memories) {
    const idx = Math.min(
      Math.floor(m.effectiveness / BUCKET_SIZE),
      BUCKET_COUNT - 1,
    );
    buckets[idx].count++;
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <BarChart data={buckets}>
        <CartesianGrid strokeDasharray="3 3" vertical={false} />
        <XAxis dataKey="range" fontSize={12} />
        <YAxis allowDecimals={false} fontSize={12} />
        <Tooltip />
        <Bar dataKey="count" fill="var(--color-primary)" radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}
