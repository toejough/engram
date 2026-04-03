import { Link, useParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { fetchMemory, ApiError } from "@/lib/api";
import { quadrantLabel, quadrantColor } from "@/lib/quadrants";

function formatDate(iso: string): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleString();
}

export default function MemoryDetail() {
  const { slug } = useParams<{ slug: string }>();

  const {
    data: memory,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["memory", slug],
    queryFn: () => fetchMemory(slug!),
    enabled: !!slug,
  });

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center text-muted-foreground">
        Loading...
      </div>
    );
  }

  if (error instanceof ApiError && error.status === 404) {
    return (
      <div className="mx-auto max-w-4xl space-y-4 p-8">
        <Link
          to="/"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Dashboard
        </Link>
        <div className="flex h-48 items-center justify-center text-muted-foreground">
          Memory not found or archived
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex h-64 items-center justify-center text-destructive">
        Failed to load memory. Is the engram server running?
      </div>
    );
  }

  if (!memory) return null;

  const totalEvaluations = memory.totalEvaluations;
  const effectivenessPercent = memory.effectiveness.toFixed(1);

  return (
    <div className="mx-auto max-w-4xl space-y-6 p-8">
      <Link
        to="/"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Dashboard
      </Link>

      <div className="flex items-center gap-3">
        <h1 className="text-3xl font-bold">{memory.slug}</h1>
        <Badge
          style={{ backgroundColor: quadrantColor(memory.quadrant) }}
          className="text-white"
        >
          {quadrantLabel(memory.quadrant)}
        </Badge>
      </div>

      {/* SBIA Sections */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm text-muted-foreground">
              Situation
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="whitespace-pre-wrap">{memory.situation}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm text-muted-foreground">
              Behavior
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="whitespace-pre-wrap">{memory.behavior}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm text-muted-foreground">
              Impact
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="whitespace-pre-wrap">{memory.impact}</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm text-muted-foreground">
              Action
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="whitespace-pre-wrap">{memory.action}</p>
          </CardContent>
        </Card>
      </div>

      {/* Effectiveness Metrics */}
      <Card>
        <CardHeader>
          <CardTitle>Effectiveness</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
            <div>
              <div className="text-sm text-muted-foreground">Effectiveness</div>
              <div className="text-2xl font-bold">{effectivenessPercent}%</div>
            </div>
            <div>
              <div className="text-sm text-muted-foreground">Followed</div>
              <div className="text-2xl font-bold">{memory.followedCount}</div>
            </div>
            <div>
              <div className="text-sm text-muted-foreground">Not Followed</div>
              <div className="text-2xl font-bold">
                {memory.notFollowedCount}
              </div>
            </div>
            <div>
              <div className="text-sm text-muted-foreground">Irrelevant</div>
              <div className="text-2xl font-bold">{memory.irrelevantCount}</div>
            </div>
            <div>
              <div className="text-sm text-muted-foreground">
                Total Evaluations
              </div>
              <div className="text-2xl font-bold">{totalEvaluations}</div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Metadata */}
      <Card>
        <CardHeader>
          <CardTitle>Metadata</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid gap-x-8 gap-y-2 sm:grid-cols-2">
            <div>
              <dt className="text-sm text-muted-foreground">Project</dt>
              <dd className="font-medium">
                {memory.projectSlug || "—"}
              </dd>
            </div>
            <div>
              <dt className="text-sm text-muted-foreground">Project Scoped</dt>
              <dd className="font-medium">
                {memory.projectScoped ? "Yes" : "No"}
              </dd>
            </div>
            <div>
              <dt className="text-sm text-muted-foreground">Created</dt>
              <dd className="font-medium">{formatDate(memory.createdAt)}</dd>
            </div>
            <div>
              <dt className="text-sm text-muted-foreground">Updated</dt>
              <dd className="font-medium">{formatDate(memory.updatedAt)}</dd>
            </div>
            <div>
              <dt className="text-sm text-muted-foreground">Surfaced Count</dt>
              <dd className="font-medium">{memory.surfacedCount}</dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      {/* Pending Evaluations */}
      <Card>
        <CardHeader>
          <CardTitle>
            Pending Evaluations ({memory.pendingEvaluations.length})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {memory.pendingEvaluations.length === 0 ? (
            <p className="text-muted-foreground">No pending evaluations</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Surfaced At</TableHead>
                  <TableHead>User Prompt</TableHead>
                  <TableHead>Session ID</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {memory.pendingEvaluations.map((pe, idx) => (
                  <TableRow key={idx}>
                    <TableCell className="whitespace-nowrap">
                      {formatDate(pe.surfacedAt)}
                    </TableCell>
                    <TableCell className="max-w-md truncate">
                      {pe.userPrompt}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {pe.sessionId}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
