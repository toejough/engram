import { useState, useCallback } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { fetchActivity } from "@/lib/api";
import ErrorState from "@/components/ErrorState";
import { ActivitySkeleton } from "@/components/skeletons";
import type { ActivityEntry } from "@/lib/types";

const PAGE_SIZE = 50;

function eventBadgeVariant(type: string): string {
  switch (type) {
    case "created":
      return "hsl(142 71% 45%)";
    case "updated":
      return "hsl(217 91% 60%)";
    case "surfaced":
      return "hsl(280 68% 60%)";
    default:
      return "hsl(0 0% 50%)";
  }
}

function formatTimestamp(iso: string): string {
  const date = new Date(iso);
  return date.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

interface EventItemProps {
  event: ActivityEntry;
}

function EventItem({ event }: EventItemProps) {
  const color = eventBadgeVariant(event.type);

  return (
    <div className="flex items-start gap-4 py-3">
      <Badge
        variant="outline"
        className="mt-0.5 shrink-0 capitalize"
        style={{ borderColor: color, color }}
      >
        {event.type}
      </Badge>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <Link
            to={`/memories/${encodeURIComponent(event.memorySlug)}`}
            className="truncate font-mono text-sm hover:underline"
          >
            {event.memorySlug}
          </Link>
          <span className="shrink-0 text-xs text-muted-foreground">
            {formatTimestamp(event.timestamp)}
          </span>
        </div>
        {event.context && (
          <p className="mt-1 truncate text-sm text-muted-foreground">
            {event.context}
          </p>
        )}
      </div>
    </div>
  );
}

export default function Activity() {
  const [pages, setPages] = useState(1);
  const [allEvents, setAllEvents] = useState<ActivityEntry[]>([]);
  const [hasMore, setHasMore] = useState(true);

  const { isLoading, isError, isFetching, refetch } = useQuery({
    queryKey: ["activity", pages],
    queryFn: async () => {
      const data = await fetchActivity(pages, PAGE_SIZE);
      setAllEvents((prev) => {
        if (pages === 1) return data;
        // Deduplicate by appending only new events.
        const existing = new Set(prev.map((e) => `${e.type}-${e.timestamp}-${e.memorySlug}`));
        const newEvents = data.filter(
          (e) => !existing.has(`${e.type}-${e.timestamp}-${e.memorySlug}`),
        );
        return [...prev, ...newEvents];
      });
      if (data.length < PAGE_SIZE) {
        setHasMore(false);
      }
      return data;
    },
    staleTime: Infinity,
  });

  const loadMore = useCallback(() => {
    setPages((prev) => prev + 1);
  }, []);

  if (isLoading && pages === 1) {
    return (
      <div className="mx-auto max-w-4xl space-y-8 p-8">
        <div>
          <h1 className="text-3xl font-bold">Activity</h1>
          <p className="mt-1 text-muted-foreground">
            Recent memory activity derived from timestamps.
          </p>
        </div>
        <ActivitySkeleton />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="mx-auto max-w-4xl space-y-8 p-8">
        <div>
          <h1 className="text-3xl font-bold">Activity</h1>
          <p className="mt-1 text-muted-foreground">
            Recent memory activity derived from timestamps.
          </p>
        </div>
        <ErrorState
          message="Failed to load activity. Is the engram server running?"
          onRetry={() => refetch()}
        />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl space-y-8 p-8">
      <div>
        <h1 className="text-3xl font-bold">Activity</h1>
        <p className="mt-1 text-muted-foreground">
          Recent memory activity derived from timestamps.
        </p>
      </div>

      {allEvents.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-16">
            <p className="text-lg font-medium">No activity yet</p>
            <p className="mt-2 max-w-md text-center text-muted-foreground">
              Activity will appear here as memories are created, updated, and
              surfaced.
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="divide-y pt-6">
            {allEvents.map((event, index) => (
              <EventItem
                key={`${event.type}-${event.timestamp}-${event.memorySlug}-${index}`}
                event={event}
              />
            ))}
          </CardContent>
        </Card>
      )}

      {hasMore && allEvents.length > 0 && (
        <div className="flex justify-center">
          <Button variant="outline" onClick={loadMore} disabled={isFetching}>
            {isFetching ? "Loading..." : "Load more"}
          </Button>
        </div>
      )}
    </div>
  );
}
