import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
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
import { Badge } from "@/components/ui/badge";
import type { Memory } from "@/lib/types";
import { quadrantLabel } from "@/lib/quadrants";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";

type SortKey =
  | "slug"
  | "quadrant"
  | "effectiveness"
  | "surfacedCount"
  | "projectSlug"
  | "updatedAt";

type SortDir = "asc" | "desc";

const QUADRANT_BADGE_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  working: "default",
  leech: "destructive",
  "hidden-gem": "secondary",
  noise: "outline",
  "insufficient-data": "outline",
};

interface MemoryTableProps {
  memories: Memory[];
  quadrantFilter: string;
  projectFilter: string;
  projects: string[];
  onQuadrantChange: (value: string) => void;
  onProjectChange: (value: string) => void;
}

function compareValues(a: Memory, b: Memory, key: SortKey): number {
  switch (key) {
    case "slug":
    case "quadrant":
    case "projectSlug":
      return a[key].localeCompare(b[key]);
    case "effectiveness":
    case "surfacedCount":
      return a[key] - b[key];
    case "updatedAt":
      return a.updatedAt.localeCompare(b.updatedAt);
  }
}

function SortIcon({ column, sortKey, sortDir }: { column: SortKey; sortKey: SortKey; sortDir: SortDir }) {
  if (column !== sortKey) return <ArrowUpDown className="ml-1 inline size-3.5 text-muted-foreground" />;
  return sortDir === "asc"
    ? <ArrowUp className="ml-1 inline size-3.5" />
    : <ArrowDown className="ml-1 inline size-3.5" />;
}

export default function MemoryTable({
  memories,
  quadrantFilter,
  projectFilter,
  projects,
  onQuadrantChange,
  onProjectChange,
}: MemoryTableProps) {
  const navigate = useNavigate();
  const [sortKey, setSortKey] = useState<SortKey>("updatedAt");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === "asc" ? "desc" : "asc"));
    } else {
      setSortKey(key);
      setSortDir("asc");
    }
  };

  const filtered = useMemo(() => {
    let result = memories;
    if (quadrantFilter) {
      result = result.filter((m) => m.quadrant === quadrantFilter);
    }
    if (projectFilter) {
      result = result.filter((m) => m.projectSlug === projectFilter);
    }
    return result;
  }, [memories, quadrantFilter, projectFilter]);

  const sorted = useMemo(() => {
    return [...filtered].sort((a, b) => {
      const cmp = compareValues(a, b, sortKey);
      return sortDir === "asc" ? cmp : -cmp;
    });
  }, [filtered, sortKey, sortDir]);

  const columns: { key: SortKey; label: string }[] = [
    { key: "slug", label: "Slug" },
    { key: "quadrant", label: "Quadrant" },
    { key: "effectiveness", label: "Effectiveness" },
    { key: "surfacedCount", label: "Surfaced" },
    { key: "projectSlug", label: "Project" },
    { key: "updatedAt", label: "Updated" },
  ];

  return (
    <div className="space-y-4">
      <div className="flex gap-3">
        <Select
          value={quadrantFilter}
          onValueChange={(val) => onQuadrantChange(!val || val === "__all__" ? "" : val)}
        >
          <SelectTrigger>
            <SelectValue placeholder="All quadrants" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">All quadrants</SelectItem>
            <SelectItem value="working">Working</SelectItem>
            <SelectItem value="leech">Leech</SelectItem>
            <SelectItem value="hidden-gem">Hidden Gem</SelectItem>
            <SelectItem value="noise">Noise</SelectItem>
            <SelectItem value="insufficient-data">Insufficient Data</SelectItem>
          </SelectContent>
        </Select>

        <Select
          value={projectFilter}
          onValueChange={(val) => onProjectChange(!val || val === "__all__" ? "" : val)}
        >
          <SelectTrigger>
            <SelectValue placeholder="All projects" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">All projects</SelectItem>
            {projects.map((p) => (
              <SelectItem key={p} value={p}>
                {p}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((col) => (
              <TableHead
                key={col.key}
                className="cursor-pointer select-none"
                onClick={() => handleSort(col.key)}
              >
                {col.label}
                <SortIcon column={col.key} sortKey={sortKey} sortDir={sortDir} />
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {sorted.length === 0 ? (
            <TableRow>
              <TableCell colSpan={columns.length} className="text-center text-muted-foreground">
                No memories found
              </TableCell>
            </TableRow>
          ) : (
            sorted.map((m) => (
              <TableRow
                key={m.slug}
                className="cursor-pointer"
                onClick={() => navigate(`/memories/${encodeURIComponent(m.slug)}`)}
              >
                <TableCell className="font-medium">{m.slug}</TableCell>
                <TableCell>
                  <Badge variant={QUADRANT_BADGE_VARIANT[m.quadrant] ?? "outline"}>
                    {quadrantLabel(m.quadrant)}
                  </Badge>
                </TableCell>
                <TableCell>{m.effectiveness.toFixed(1)}%</TableCell>
                <TableCell>{m.surfacedCount}</TableCell>
                <TableCell>{m.projectSlug || "—"}</TableCell>
                <TableCell>
                  {new Date(m.updatedAt).toLocaleDateString()}
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}
