import { useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { fetchProjects, fetchSurface } from "@/lib/api";
import ErrorState from "@/components/ErrorState";
import { SurfaceResultsSkeleton } from "@/components/skeletons";
import type { SurfaceMatch, SurfaceNearMiss, SurfaceResult } from "@/lib/types";

const MAX_PREVIEW_LENGTH = 120;

function truncate(text: string): string {
  if (text.length <= MAX_PREVIEW_LENGTH) return text;
  return text.slice(0, MAX_PREVIEW_LENGTH) + "…";
}

export default function Surface() {
  const [query, setQuery] = useState("");
  const [project, setProject] = useState("");
  const [submittedQuery, setSubmittedQuery] = useState("");
  const [submittedProject, setSubmittedProject] = useState("");

  const projectsQuery = useQuery({
    queryKey: ["projects"],
    queryFn: fetchProjects,
  });

  const surfaceQuery = useQuery({
    queryKey: ["surface", submittedQuery, submittedProject],
    queryFn: () => fetchSurface(submittedQuery, submittedProject || undefined),
    enabled: submittedQuery.length > 0,
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = query.trim();
    if (!trimmed) return;
    setSubmittedQuery(trimmed);
    setSubmittedProject(project);
  }

  const projects = projectsQuery.data ?? [];
  const result: SurfaceResult | undefined = surfaceQuery.data;
  const hasSubmitted = submittedQuery.length > 0;

  return (
    <div className="mx-auto max-w-5xl space-y-6 p-8">
      <div>
        <h1 className="text-3xl font-bold">Surface Simulator</h1>
        <p className="mt-1 text-muted-foreground">
          Test which memories would surface for a given query. Enter a prompt
          to see matched memories ranked by BM25 and final scores, plus
          near-misses that fell just below the threshold.
        </p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <form onSubmit={handleSubmit} className="flex gap-3">
            <Input
              placeholder="Enter a query to simulate surfacing…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="flex-1"
            />
            <Select
              value={project}
              onValueChange={(value) => setProject(value === "__all__" ? "" : (value ?? ""))}
            >
              <SelectTrigger className="w-48">
                <SelectValue placeholder="All projects" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">All projects</SelectItem>
                {projects.map((p) => (
                  <SelectItem key={p.projectSlug} value={p.projectSlug}>
                    {p.projectSlug}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button type="submit" disabled={!query.trim() || surfaceQuery.isFetching}>
              <Search className="mr-1.5 h-4 w-4" />
              Simulate
            </Button>
          </form>
        </CardContent>
      </Card>

      {!hasSubmitted && (
        <div className="flex h-48 items-center justify-center text-muted-foreground">
          Enter a query above to see which memories would be surfaced.
        </div>
      )}

      {surfaceQuery.isFetching && (
        <SurfaceResultsSkeleton />
      )}

      {surfaceQuery.isError && !surfaceQuery.isFetching && (
        <ErrorState
          message="Failed to run simulation. Is the engram server running?"
          onRetry={() => surfaceQuery.refetch()}
        />
      )}

      {result && !surfaceQuery.isFetching && (
        <>
          <MatchesSection matches={result.matches} />
          <NearMissesSection nearMisses={result.nearMisses} />
        </>
      )}
    </div>
  );
}

function MatchesSection({ matches }: { matches: SurfaceMatch[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          Matches
          <Badge variant="secondary">{matches.length}</Badge>
        </CardTitle>
      </CardHeader>
      <CardContent>
        {matches.length === 0 ? (
          <p className="text-muted-foreground">
            No memories matched above the threshold for this query.
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Memory</TableHead>
                <TableHead className="w-24 text-right">BM25</TableHead>
                <TableHead className="w-24 text-right">Final</TableHead>
                <TableHead>SBIA Preview</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {matches.map((match) => (
                <MatchRow key={match.slug} match={match} />
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

function NearMissesSection({ nearMisses }: { nearMisses: SurfaceNearMiss[] }) {
  if (nearMisses.length === 0) return null;

  const threshold = nearMisses[0].threshold;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          Near Misses
          <Badge variant="outline">{nearMisses.length}</Badge>
          <span className="text-sm font-normal text-muted-foreground">
            Below threshold ({threshold.toFixed(2)})
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Memory</TableHead>
              <TableHead className="w-24 text-right">BM25</TableHead>
              <TableHead className="w-24 text-right">Final</TableHead>
              <TableHead>SBIA Preview</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {nearMisses.map((miss) => (
              <MatchRow key={miss.slug} match={miss} />
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

function MatchRow({ match }: { match: SurfaceMatch }) {
  const preview = [
    match.situation ? `S: ${truncate(match.situation)}` : "",
    match.action ? `A: ${truncate(match.action)}` : "",
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <TableRow>
      <TableCell>
        <Link
          to={`/memories/${encodeURIComponent(match.slug)}`}
          className="font-medium text-primary hover:underline"
        >
          {match.slug}
        </Link>
      </TableCell>
      <TableCell className="text-right font-mono text-sm">
        {match.bm25Score.toFixed(3)}
      </TableCell>
      <TableCell className="text-right font-mono text-sm">
        {match.finalScore.toFixed(3)}
      </TableCell>
      <TableCell className="max-w-md text-sm text-muted-foreground">
        {preview || "—"}
      </TableCell>
    </TableRow>
  );
}
