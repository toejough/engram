import { useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { fetchProjects, fetchMemories } from "@/lib/api";
import { quadrantLabel, quadrantColor } from "@/lib/quadrants";
import ErrorState from "@/components/ErrorState";
import { ProjectCardsSkeleton } from "@/components/skeletons";
import type { Memory, Project } from "@/lib/types";

const QUADRANT_ORDER = [
  "working",
  "leech",
  "hidden-gem",
  "noise",
  "insufficient-data",
];

const MAX_LEECHES = 5;

function getTopLeeches(memories: Memory[], projectSlug: string): Memory[] {
  return memories
    .filter(
      (m) =>
        m.quadrant === "leech" &&
        (projectSlug === "" ? m.projectSlug === "" : m.projectSlug === projectSlug),
    )
    .sort((a, b) => b.surfacedCount - a.surfacedCount)
    .slice(0, MAX_LEECHES);
}

interface ProjectCardProps {
  project: Project;
  leeches: Memory[];
  onClick: () => void;
}

function ProjectCard({ project, leeches, onClick }: ProjectCardProps) {
  return (
    <Card
      className="cursor-pointer transition-colors hover:bg-muted/50"
      onClick={onClick}
    >
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span className="font-mono text-lg">
            {project.projectSlug || "(no project)"}
          </span>
          <span className="text-sm font-normal text-muted-foreground">
            {project.memoryCount} {project.memoryCount === 1 ? "memory" : "memories"}
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center gap-4">
          <div className="text-sm text-muted-foreground">Avg Effectiveness</div>
          <div className="text-lg font-semibold">
            {project.avgEffectiveness.toFixed(1)}%
          </div>
        </div>

        <div>
          <div className="mb-2 text-sm text-muted-foreground">Quadrant Breakdown</div>
          <div className="flex flex-wrap gap-2">
            {QUADRANT_ORDER.map((q) => {
              const count = project.quadrantBreakdown[q] ?? 0;
              if (count === 0) return null;
              return (
                <Badge
                  key={q}
                  variant="outline"
                  style={{ borderColor: quadrantColor(q), color: quadrantColor(q) }}
                >
                  {quadrantLabel(q)}: {count}
                </Badge>
              );
            })}
          </div>
        </div>

        {leeches.length > 0 && (
          <div>
            <div className="mb-2 text-sm text-muted-foreground">
              Top Leeches
            </div>
            <ul className="space-y-1">
              {leeches.map((m) => (
                <li
                  key={m.slug}
                  className="flex items-center justify-between text-sm"
                >
                  <span className="truncate font-mono text-destructive">
                    {m.slug}
                  </span>
                  <span className="ml-2 shrink-0 text-muted-foreground">
                    surfaced {m.surfacedCount}x
                  </span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

export default function Projects() {
  const navigate = useNavigate();

  const projectsQuery = useQuery({
    queryKey: ["projects"],
    queryFn: fetchProjects,
  });

  const memoriesQuery = useQuery({
    queryKey: ["memories"],
    queryFn: fetchMemories,
  });

  const leechesByProject = useMemo(() => {
    if (!memoriesQuery.data) return new Map<string, Memory[]>();
    const map = new Map<string, Memory[]>();
    const projects = projectsQuery.data ?? [];
    for (const project of projects) {
      map.set(project.projectSlug, getTopLeeches(memoriesQuery.data, project.projectSlug));
    }
    return map;
  }, [memoriesQuery.data, projectsQuery.data]);

  const isLoading = projectsQuery.isLoading || memoriesQuery.isLoading;
  const isError = projectsQuery.isError || memoriesQuery.isError;

  if (isError) {
    return (
      <div className="mx-auto max-w-7xl space-y-8 p-8">
        <div>
          <h1 className="text-3xl font-bold">Projects</h1>
          <p className="mt-1 text-muted-foreground">
            Memory breakdown by project.
          </p>
        </div>
        <ErrorState
          message="Failed to load data. Is the engram server running?"
          onRetry={() => {
            projectsQuery.refetch();
            memoriesQuery.refetch();
          }}
        />
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="mx-auto max-w-7xl space-y-8 p-8">
        <div>
          <h1 className="text-3xl font-bold">Projects</h1>
          <p className="mt-1 text-muted-foreground">
            Memory breakdown by project.
          </p>
        </div>
        <ProjectCardsSkeleton />
      </div>
    );
  }

  const projects = projectsQuery.data!;

  if (projects.length === 0) {
    return (
      <div className="mx-auto max-w-7xl space-y-8 p-8">
        <div>
          <h1 className="text-3xl font-bold">Projects</h1>
          <p className="mt-1 text-muted-foreground">
            Memory breakdown by project.
          </p>
        </div>
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <p className="text-lg font-medium">No projects yet</p>
            <p className="mt-2 max-w-md text-center text-muted-foreground">
              Projects appear here when memories are tagged with a project slug.
            </p>
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl space-y-8 p-8">
      <div>
        <h1 className="text-3xl font-bold">Projects</h1>
        <p className="mt-1 text-muted-foreground">
          Memory breakdown by project.
        </p>
      </div>

      <div className="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
        {projects.map((project) => (
          <ProjectCard
            key={project.projectSlug}
            project={project}
            leeches={leechesByProject.get(project.projectSlug) ?? []}
            onClick={() =>
              navigate(`/?project=${encodeURIComponent(project.projectSlug)}`)
            }
          />
        ))}
      </div>
    </div>
  );
}
