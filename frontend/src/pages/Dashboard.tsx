import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { fetchMemories, fetchStats } from "@/lib/api";
import StatCards from "@/components/StatCards";
import QuadrantChart from "@/components/QuadrantChart";
import EffectivenessHistogram from "@/components/EffectivenessHistogram";
import MemoryTable from "@/components/MemoryTable";

export default function Dashboard() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialProject = searchParams.get("project") ?? "";

  const [quadrantFilter, setQuadrantFilter] = useState("");
  const [projectFilter, setProjectFilter] = useState(initialProject);

  const memoriesQuery = useQuery({
    queryKey: ["memories"],
    queryFn: fetchMemories,
  });

  const statsQuery = useQuery({
    queryKey: ["stats"],
    queryFn: fetchStats,
  });

  const projects = useMemo(() => {
    if (!memoriesQuery.data) return [];
    const slugs = new Set(
      memoriesQuery.data
        .map((m) => m.projectSlug)
        .filter((s) => s.length > 0),
    );
    return [...slugs].sort();
  }, [memoriesQuery.data]);

  const handleProjectChange = (value: string) => {
    setProjectFilter(value);
    if (value) {
      setSearchParams({ project: value });
    } else {
      setSearchParams({});
    }
  };

  if (memoriesQuery.isLoading || statsQuery.isLoading) {
    return (
      <div className="flex h-64 items-center justify-center text-muted-foreground">
        Loading...
      </div>
    );
  }

  if (memoriesQuery.isError || statsQuery.isError) {
    return (
      <div className="flex h-64 items-center justify-center text-destructive">
        Failed to load data. Is the engram server running?
      </div>
    );
  }

  const memories = memoriesQuery.data!;
  const stats = statsQuery.data!;

  return (
    <div className="mx-auto max-w-7xl space-y-8 p-8">
      <div>
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="mt-1 text-muted-foreground">
          Memory overview and statistics
        </p>
      </div>

      <StatCards stats={stats} />

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Quadrant Distribution</CardTitle>
          </CardHeader>
          <CardContent>
            <QuadrantChart stats={stats} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Effectiveness Distribution</CardTitle>
          </CardHeader>
          <CardContent>
            <EffectivenessHistogram memories={memories} />
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Memories</CardTitle>
        </CardHeader>
        <CardContent>
          <MemoryTable
            memories={memories}
            quadrantFilter={quadrantFilter}
            projectFilter={projectFilter}
            projects={projects}
            onQuadrantChange={setQuadrantFilter}
            onProjectChange={handleProjectChange}
          />
        </CardContent>
      </Card>
    </div>
  );
}
