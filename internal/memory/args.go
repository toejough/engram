package memory

import "time"

// ArchiveListArgs holds the command-line arguments for memory archive list.
type ArchiveListArgs struct {
	MemoryRoot string `targ:"flag,desc=Path to memory root directory"`
	Limit      int    `targ:"flag,short=n,desc=Maximum entries to show (default 50)"`
}

// DecideArgs holds the command-line arguments for memory decide.
type DecideArgs struct {
	Context      string `targ:"flag,short=c,required,desc=Decision context"`
	Choice       string `targ:"flag,required,desc=The choice made"`
	Reason       string `targ:"flag,short=r,required,desc=Reason for the decision"`
	Alternatives string `targ:"flag,short=a,desc=Comma-separated alternatives considered"`
	Project      string `targ:"flag,short=p,required,desc=Project name"`
	MemoryRoot   string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

// DiagArgs holds the command-line arguments for memory diag.
type DiagArgs struct {
	MemoryRoot string `targ:"flag,name=memory-root,desc=Memory root directory (default: ~/.claude/memory)"`
}

// DiagnoseArgs holds the command-line arguments for memory diagnose.
type DiagnoseArgs struct {
	MemoryRoot string `targ:"flag,name=memory-root,desc=Memory root directory (default: ~/.claude/memory)"`
	ID         int64  `targ:"flag,name=id,desc=Diagnose a specific memory by ID (default: all leeches)"`
	NoLLM      bool   `targ:"flag,name=no-llm,desc=Skip LLM-based rewrite preview for content_quality diagnoses"`
	NoSave     bool   `targ:"flag,name=no-save,desc=Do not save recommendations to file"`
}

// DigestArgs holds the command-line arguments for memory digest.
type DigestArgs struct {
	Since      string `targ:"flag,short=s,desc=Time window like 7d or 24h or 168h,default=168h"`
	Tier       string `targ:"flag,short=t,desc=Filter by tier: skill or embedding or claude_md"`
	FlagsOnly  bool   `targ:"flag,short=f,desc=Show only flags not full digest"`
	MaxEntries int    `targ:"flag,short=n,desc=Maximum number of entries to show"`
	MemoryRoot string `targ:"flag,desc=Memory root directory"`
}

// ExtractArgs holds the command-line arguments for memory extract.
type ExtractArgs struct {
	Result     string `targ:"flag,short=r,desc=Path to result.toml file"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
	ModelDir   string `targ:"flag,desc=Model directory (defaults to ~/.claude/models)"`
}

// ExtractSessionArgs holds the command-line arguments for extract-session.
type ExtractSessionArgs struct {
	TranscriptPath string        `targ:"flag,name=transcript,desc=Path to JSONL transcript file"`
	MemoryRoot     string        `targ:"flag,name=memory-root,desc=Memory root directory (default: ~/.claude/memory)"`
	Project        string        `targ:"flag,name=project,desc=Project name for tagging extracted learnings (default: derived from stdin cwd)"`
	Timeout        time.Duration // Timeout duration (default: 60s). Exported for testing.
}

// FeedbackArgs holds the command-line arguments for memory feedback.
type FeedbackArgs struct {
	ID         int64  `targ:"flag,name=id,desc=Memory ID to give feedback on (ID-based mode)"`
	Helpful    bool   `targ:"flag,desc=Mark memory as helpful"`
	Wrong      bool   `targ:"flag,desc=Mark memory as wrong or not useful"`
	Unclear    bool   `targ:"flag,desc=Mark memory as unclear"`
	SessionID  string `targ:"flag,name=session-id,desc=Session ID for session-based feedback"`
	Type       string `targ:"flag,name=type,desc=Feedback type: helpful|wrong|unclear (required with --session-id)"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

// GrepArgs holds the command-line arguments for memory grep.
type GrepArgs struct {
	Pattern          string `targ:"positional,required,desc=Pattern to search for"`
	Project          string `targ:"flag,short=p,desc=Limit search to specific project"`
	IncludeDecisions bool   `targ:"flag,short=d,desc=Also search decisions files"`
	MemoryRoot       string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

// HooksCheckClaudeMDArgs holds the command-line arguments for hooks check-claudemd.
type HooksCheckClaudeMDArgs struct {
	ClaudeMDPath string `targ:"flag,desc=Path to CLAUDE.md (default: ~/.claude/CLAUDE.md)"`
	MaxLines     int    `targ:"flag,desc=Maximum line count (default: 260)"`
}

// HooksCheckEmbeddingArgs holds the command-line arguments for hooks check-embedding.
type HooksCheckEmbeddingArgs struct {
	MemoryRoot string `targ:"flag,desc=Path to memory root directory (default: ~/.claude/memory)"`
}

// HooksCheckSkillArgs holds the command-line arguments for hooks check-skill.
type HooksCheckSkillArgs struct {
	SkillsDir string `targ:"flag,desc=Path to skills directory (default: ~/.claude/skills)"`
}

// HooksInstallArgs holds the command-line arguments for hooks install.
type HooksInstallArgs struct {
	SettingsPath string `targ:"flag,desc=Path to Claude Code settings.json (default: ~/.claude/settings.json)"`
}

// HooksShowArgs holds the command-line arguments for hooks show.
type HooksShowArgs struct {
	SettingsPath string `targ:"flag,desc=Path to Claude Code settings.json (default: ~/.claude/settings.json)"`
}

// HooksStatsArgs holds the command-line arguments for hooks stats.
type HooksStatsArgs struct {
	MemoryRoot string `targ:"flag,desc=Path to memory root directory (default: ~/.claude/memory)"`
}

// LearnArgs holds the command-line arguments for memory learn.
type LearnArgs struct {
	Message    string `targ:"flag,short=m,required,desc=Learning message to store"`
	Project    string `targ:"flag,short=p,desc=Project to tag the learning with"`
	Source     string `targ:"flag,short=s,desc=Source type: internal or external (default: internal)"`
	Type       string `targ:"flag,short=t,desc=Memory type: correction or reflection (default: empty)"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
	NoLLM      bool   `targ:"flag,desc=Disable LLM-based knowledge extraction"`
}

// LearnSessionsArgs holds the command-line arguments for learn-sessions.
type LearnSessionsArgs struct {
	Days       int    `targ:"flag,name=days,desc=Look back N days (default: 7),default=7"`
	Last       int    `targ:"flag,name=last,desc=Process only last N sessions (overrides --days)"`
	MinSize    string `targ:"flag,name=min-size,desc=Minimum session size (default: 8KB),default=8KB"`
	DryRun     bool   `targ:"flag,name=dry-run,desc=List sessions without processing"`
	ResetLast  int    `targ:"flag,name=reset-last,desc=Reset last N processed sessions and exit"`
	MemoryRoot string `targ:"flag,name=memory-root,desc=Memory root directory (default: ~/.claude/memory)"`
}

// OptimizeArgs holds the command-line arguments for memory optimize.
type OptimizeArgs struct {
	Review                   bool    `targ:"flag,desc=Use interactive proposal review mode (new ISSUE-212 workflow)"`
	Yes                      bool    `targ:"flag,short=y,desc=Auto-approve all interactive prompts"`
	ClaudeMD                 string  `targ:"flag,name=claude-md,desc=Path to CLAUDE.md (defaults to ~/.claude/CLAUDE.md)"`
	MemoryRoot               string  `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
	SkillPromotionThreshold  float64 `targ:"flag,desc=Minimum utility threshold for skill promotion to CLAUDE.md (default 0.8)"`
	SkillDemotionThreshold   float64 `targ:"flag,desc=Utility threshold for skill demotion/pruning (default 0.4)"`
	MinSkillProjects         int     `targ:"flag,desc=Minimum number of projects for skill promotion (default 3)"`
	MinSkillConfidenceThresh float64 `targ:"flag,name=min-skill-confidence,desc=Minimum confidence for skill promotion (default 0.8)"`
	ForceReorg               bool    `targ:"flag,desc=Force full skill reorganization regardless of last run time (normally runs every 30 days)"`
	NoLLM                    bool    `targ:"flag,desc=Disable all LLM-based features (extractor / specificity detector / skill compiler)"`
	Tier                     string  `targ:"flag,desc=Filter proposals by tier: embeddings / skills / claude-md (ISSUE-184)"`
	NoTestSkills             bool    `targ:"flag,desc=Disable skill testing before deployment (default is to test; Task 8)"`
	TestRuns                 int     `targ:"flag,desc=Number of test runs for RED/GREEN protocol (default 3; Task 8)"`
	NoLLMEval                bool    `targ:"flag,name=no-llm-eval,desc=Skip LLM triage and behavioral testing (mechanical + human only)"`
}

// QueryArgs holds the command-line arguments for memory query.
type QueryArgs struct {
	Text                string  `targ:"positional,desc=Text to search for"`
	Limit               int     `targ:"flag,short=n,desc=Maximum number of results (default 10)"`
	Project             string  `targ:"flag,short=p,desc=Project name for retrieval tracking"`
	Verbose             bool    `targ:"flag,short=v,desc=Show detailed scoring info"`
	MemoryRoot          string  `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
	MinConfidence       int     `targ:"flag,name=min-confidence,desc=Minimum confidence threshold 0-100 (default: 0)"`
	SimilarityThreshold float64 `targ:"flag,name=similarity-threshold,desc=Minimum similarity score 0.0-1.0 (default: 0.7)"`
	MaxTokens           int     `targ:"flag,name=max-tokens,desc=Max token count for output (default: 2000)"`
	Primacy             bool    `targ:"flag,desc=Sort corrections first (primacy ordering)"`
	Rich                bool    `targ:"flag,desc=Show full metadata (confidence/retrieval count/match type/projects)"`
	Curate              bool    `targ:"flag,desc=Use LLM curation for result selection and relevance annotations"`
	StdinProject        bool    `targ:"flag,name=stdin-project,desc=Derive project from stdin hook JSON cwd"`
	StdinPrompt         bool    `targ:"flag,name=stdin-prompt,desc=Read query text and project from stdin hook JSON prompt field"`
	StdinTool           bool    `targ:"flag,name=stdin-tool,desc=Read query from stdin hook JSON tool_name + tool_input fields"`
}

// ScoreSessionArgs holds the command-line arguments for score-session.
type ScoreSessionArgs struct {
	MemoryRoot string        `targ:"flag,name=memory-root,desc=Memory root directory (default: ~/.claude/memory)"`
	Timeout    time.Duration // Timeout duration (default: 60s). Exported for testing.
}

// SkillFeedbackArgs holds the command-line arguments for memory skill feedback.
type SkillFeedbackArgs struct {
	Skill      string `targ:"flag,required,desc=Skill slug to provide feedback for"`
	Success    bool   `targ:"flag,desc=Record positive feedback (increases confidence)"`
	Failure    bool   `targ:"flag,desc=Record negative feedback (decreases confidence)"`
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}

// SkillListArgs holds the command-line arguments for memory skill list.
type SkillListArgs struct {
	MemoryRoot string `targ:"flag,desc=Memory root directory (defaults to ~/.claude/memory)"`
}
