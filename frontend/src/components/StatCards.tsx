import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Stats } from "@/lib/types";
import { quadrantLabel } from "@/lib/quadrants";

const QUADRANT_ORDER = [
  "working",
  "leech",
  "hidden-gem",
  "noise",
  "insufficient-data",
];

interface StatCardsProps {
  stats: Stats;
}

export default function StatCards({ stats }: StatCardsProps) {
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <Card>
        <CardHeader>
          <CardTitle className="text-sm text-muted-foreground">
            Total Memories
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-3xl font-bold">{stats.totalMemories}</div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-sm text-muted-foreground">
            Avg Effectiveness
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-3xl font-bold">
            {stats.avgEffectiveness.toFixed(1)}%
          </div>
        </CardContent>
      </Card>

      {QUADRANT_ORDER.map((q) => (
        <Card key={q}>
          <CardHeader>
            <CardTitle className="text-sm text-muted-foreground">
              {quadrantLabel(q)}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">
              {stats.quadrantDistribution[q] ?? 0}
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
