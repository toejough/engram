export interface Memory {
  slug: string;
  situation: string;
  behavior: string;
  impact: string;
  action: string;
  projectScoped: boolean;
  projectSlug: string;
  surfacedCount: number;
  followedCount: number;
  notFollowedCount: number;
  irrelevantCount: number;
  effectiveness: number;
  quadrant: string;
  totalEvaluations: number;
  updatedAt: string;
}

export interface PendingEvaluation {
  surfacedAt: string;
  userPrompt: string;
  sessionId: string;
  projectSlug: string;
}

export interface MemoryDetail extends Memory {
  pendingEvaluations: PendingEvaluation[];
}

export interface Stats {
  totalMemories: number;
  avgEffectiveness: number;
  quadrantDistribution: Record<string, number>;
}

export interface Project {
  projectSlug: string;
  memoryCount: number;
  avgEffectiveness: number;
  quadrantBreakdown: Record<string, number>;
}

export interface SurfaceMatch {
  slug: string;
  bm25Score: number;
  finalScore: number;
  situation: string;
  action: string;
}

export interface SurfaceNearMiss extends SurfaceMatch {
  threshold: number;
}

export interface SurfaceResult {
  matches: SurfaceMatch[];
  nearMisses: SurfaceNearMiss[];
}

export interface ActivityEntry {
  type: string;
  timestamp: string;
  memorySlug: string;
  context: string;
}
