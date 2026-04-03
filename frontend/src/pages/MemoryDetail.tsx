import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Pencil } from "lucide-react";
import { toast } from "sonner";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { fetchMemory, updateMemory, ApiError } from "@/lib/api";
import { quadrantLabel, quadrantColor } from "@/lib/quadrants";

function formatDate(iso: string): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleString();
}

interface EditFormState {
  situation: string;
  behavior: string;
  impact: string;
  action: string;
  projectScoped: boolean;
  projectSlug: string;
}

export default function MemoryDetail() {
  const { slug } = useParams<{ slug: string }>();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form, setForm] = useState<EditFormState>({
    situation: "",
    behavior: "",
    impact: "",
    action: "",
    projectScoped: false,
    projectSlug: "",
  });

  const {
    data: memory,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["memory", slug],
    queryFn: () => fetchMemory(slug!),
    enabled: !!slug,
  });

  function enterEditMode() {
    if (!memory) return;
    setForm({
      situation: memory.situation,
      behavior: memory.behavior,
      impact: memory.impact,
      action: memory.action,
      projectScoped: memory.projectScoped,
      projectSlug: memory.projectSlug,
    });
    setEditing(true);
  }

  function cancelEdit() {
    setEditing(false);
  }

  async function saveEdit() {
    if (!slug) return;
    setSaving(true);
    try {
      await updateMemory(slug, form);
      await queryClient.invalidateQueries({ queryKey: ["memory", slug] });
      setEditing(false);
      toast.success("Memory updated successfully");
    } catch (err) {
      const message =
        err instanceof ApiError ? err.message : "Failed to update memory";
      toast.error(message);
    } finally {
      setSaving(false);
    }
  }

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
        {!editing && (
          <Button variant="outline" size="sm" onClick={enterEditMode}>
            <Pencil className="mr-1 h-3.5 w-3.5" />
            Edit
          </Button>
        )}
      </div>

      {/* SBIA Sections */}
      {editing ? (
        <div className="space-y-6">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="situation">Situation</Label>
              <Textarea
                id="situation"
                placeholder="When does this apply? What context triggers this memory?"
                rows={5}
                value={form.situation}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, situation: e.target.value }))
                }
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="behavior">Behavior</Label>
              <Textarea
                id="behavior"
                placeholder="What should the agent do? Specific actions or patterns to follow."
                rows={5}
                value={form.behavior}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, behavior: e.target.value }))
                }
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="impact">Impact</Label>
              <Textarea
                id="impact"
                placeholder="Why does this matter? What goes wrong without this memory?"
                rows={5}
                value={form.impact}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, impact: e.target.value }))
                }
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="action">Action</Label>
              <Textarea
                id="action"
                placeholder="Concrete steps or rules. What exactly should be done?"
                rows={5}
                value={form.action}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, action: e.target.value }))
                }
              />
            </div>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Project Settings</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-2">
                <Checkbox
                  id="projectScoped"
                  checked={form.projectScoped}
                  onCheckedChange={(checked) =>
                    setForm((prev) => ({
                      ...prev,
                      projectScoped: checked === true,
                    }))
                  }
                />
                <Label htmlFor="projectScoped">Project scoped</Label>
              </div>
              <div className="max-w-sm space-y-2">
                <Label htmlFor="projectSlug">Project slug</Label>
                <Input
                  id="projectSlug"
                  placeholder="e.g. my-project"
                  value={form.projectSlug}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      projectSlug: e.target.value,
                    }))
                  }
                />
              </div>
            </CardContent>
          </Card>

          <div className="flex gap-2">
            <Button onClick={saveEdit} disabled={saving}>
              {saving ? "Saving..." : "Save"}
            </Button>
            <Button variant="outline" onClick={cancelEdit} disabled={saving}>
              Cancel
            </Button>
          </div>
        </div>
      ) : (
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
      )}

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
