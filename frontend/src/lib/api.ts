import type {
  Memory,
  MemoryDetail,
  Stats,
  Project,
  SurfaceResult,
  ActivityEntry,
} from "./types";

const BASE = "/api";

async function fetchJSON<T>(path: string): Promise<T> {
  const response = await fetch(`${BASE}${path}`);
  if (!response.ok) {
    throw new Error(`API error: ${response.status} ${response.statusText}`);
  }
  return response.json() as Promise<T>;
}

export function fetchMemories(): Promise<Memory[]> {
  return fetchJSON<Memory[]>("/memories");
}

export function fetchMemory(slug: string): Promise<MemoryDetail> {
  return fetchJSON<MemoryDetail>(`/memories/${encodeURIComponent(slug)}`);
}

export function fetchStats(): Promise<Stats> {
  return fetchJSON<Stats>("/stats");
}

export function fetchProjects(): Promise<Project[]> {
  return fetchJSON<Project[]>("/projects");
}

export function fetchSurface(
  query: string,
  project?: string,
): Promise<SurfaceResult> {
  const params = new URLSearchParams({ q: query });
  if (project) {
    params.set("project", project);
  }
  return fetchJSON<SurfaceResult>(`/surface?${params.toString()}`);
}

export function fetchActivity(
  page: number,
  limit: number = 50,
): Promise<ActivityEntry[]> {
  const params = new URLSearchParams({
    page: String(page),
    limit: String(limit),
  });
  return fetchJSON<ActivityEntry[]>(`/activity?${params.toString()}`);
}
