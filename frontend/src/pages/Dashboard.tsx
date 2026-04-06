import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { fetchMemories, fetchStats } from "@/lib/api";
import StatCards from "@/components/StatCards";
import QuadrantChart from "@/components/QuadrantChart";
import EffectivenessHistogram from "@/components/EffectivenessHistogram";
import MemoryTable from "@/components/MemoryTable";
import ErrorState from "@/components/ErrorState";
import {
  StatCardsSkeleton,
  ChartSkeleton,
  TableRowsSkeleton,
} from "@/components/skeletons";
import {
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
} from "@/components/ui/table";

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

  const isLoading = memoriesQuery.isLoading || statsQuery.isLoading;
  const isError = memoriesQuery.isError || statsQuery.isError;

  if (isError) {
    return (
      <div className="mx-auto max-w-7xl space-y-8 p-8">
        <div>
          <h1 className="text-3xl font-bold">Dashboard</h1>
          <p className="mt-1 text-muted-foreground">
            Memory overview and statistics
          </p>
        </div>
        <ErrorState
          message="Failed to load data. Is the engram server running?"
          onRetry={() => {
            memoriesQuery.refetch();
            statsQuery.refetch();
          }}
        />
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="mx-auto max-w-7xl space-y-8 p-8">
        <div>
          <h1 className="text-3xl font-bold">Dashboard</h1>
          <p className="mt-1 text-muted-foreground">
            Memory overview and statistics
          </p>
        </div>
        <StatCardsSkeleton />
        <div className="grid gap-6 lg:grid-cols-2">
          <ChartSkeleton />
          <ChartSkeleton />
        </div>
        <Card>
          <CardHeader>
            <CardTitle>Memories</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Slug</TableHead>
                  <TableHead>Quadrant</TableHead>
                  <TableHead>Effectiveness</TableHead>
                  <TableHead>Surfaced</TableHead>
                  <TableHead>Project</TableHead>
                  <TableHead>Updated</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRowsSkeleton />
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    );
  }

  const memories = memoriesQuery.data!;
  const stats = statsQuery.data!;

  if (memories.length === 0) {
    return (
      <div className="mx-auto max-w-7xl space-y-8 p-8">
        <div>
          <h1 className="text-3xl font-bold">Dashboard</h1>
          <p className="mt-1 text-muted-foreground">
            Memory overview and statistics
          </p>
        </div>

        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <p className="text-lg font-medium">No memories yet</p>
            <p className="mt-2 max-w-md text-center text-muted-foreground">
              Memories are created automatically as you use Claude Code with
              engram installed.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

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
