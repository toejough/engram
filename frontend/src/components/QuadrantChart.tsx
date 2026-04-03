import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer } from "recharts";
import type { Stats } from "@/lib/types";
import { quadrantColor, quadrantLabel } from "@/lib/quadrants";

interface QuadrantChartProps {
  stats: Stats;
}

export default function QuadrantChart({ stats }: QuadrantChartProps) {
  const data = Object.entries(stats.quadrantDistribution)
    .filter(([, count]) => count > 0)
    .map(([quadrant, count]) => ({
      name: quadrantLabel(quadrant),
      value: count,
      color: quadrantColor(quadrant),
    }));

  if (data.length === 0) {
    return (
      <div className="flex h-64 items-center justify-center text-muted-foreground">
        No data to display
      </div>
    );
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <PieChart>
        <Pie
          data={data}
          dataKey="value"
          nameKey="name"
          cx="50%"
          cy="50%"
          innerRadius={60}
          outerRadius={100}
          paddingAngle={2}
        >
          {data.map((entry) => (
            <Cell key={entry.name} fill={entry.color} />
          ))}
        </Pie>
        <Tooltip />
        <Legend />
      </PieChart>
    </ResponsiveContainer>
  );
}
