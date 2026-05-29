//go:build targ

package eval

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Exported variables.
var (
	ErrUnknownArm = errors.New("unknown arm")
)

// Run executes every scenario under the named arm and writes results.
// Orchestration only; all I/O goes through deps.
func Run(ctx context.Context, armName string, cfg RunConfig, deps Deps) error {
	arm, ok := LookupArm(armName)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownArm, armName)
	}

	root, err := os.MkdirTemp("", "engram-eval-"+arm.Name+"-")
	if err != nil {
		return fmt.Errorf("creating run root: %w", err)
	}

	configDir, pathPrefix, err := deps.Config.Build(ctx, arm, root)
	if err != nil {
		return fmt.Errorf("building arm config: %w", err)
	}

	trials := cfg.Trials
	if trials < 1 {
		trials = 1
	}

	for _, scenario := range Scenarios() {
		for trial := range trials {
			result, runErr := runOne(ctx, arm, scenario, trial, cfg, configDir, pathPrefix, root, deps)
			if runErr != nil {
				return fmt.Errorf("scenario %q trial %d: %w", scenario.Name, trial, runErr)
			}

			if err := deps.Results.Append(ctx, result); err != nil {
				return fmt.Errorf("writing result: %w", err)
			}
		}
	}

	return nil
}

func runOne(
	ctx context.Context, arm Arm, scenario Scenario, trial int, cfg RunConfig,
	configDir, pathPrefix, root string, deps Deps,
) (RunResult, error) {
	workspace, err := os.MkdirTemp(root, fmt.Sprintf("ws-%s-%d-", scenario.Name, trial))
	if err != nil {
		return RunResult{}, fmt.Errorf("creating workspace: %w", err)
	}

	vaultDir := filepath.Join(workspace, ".vault")

	if err := deps.Cloner.Clone(ctx, cfg.VaultSrc, vaultDir); err != nil {
		return RunResult{}, fmt.Errorf("cloning vault: %w", err)
	}

	res, err := deps.Runner.Run(ctx, AgentInvocation{
		Prompt:     scenario.Prompt,
		Model:      cfg.Model,
		Workspace:  workspace,
		ConfigDir:  configDir,
		PathPrefix: pathPrefix,
		VaultPath:  vaultDir,
	})
	if err != nil {
		return RunResult{}, fmt.Errorf("running agent: %w", err)
	}

	rs, err := ParseResult(res.ResultJSON)
	if err != nil {
		return RunResult{}, fmt.Errorf("parsing result: %w", err)
	}

	cmds := ParseBashCommands(res.TranscriptRaw)

	return RunResult{
		Arm:       arm.Name,
		Scenario:  scenario.Name,
		Trial:     trial,
		Layer1:    rs.Layer1(),
		Behaviors: DetectBehaviors(scenario, cmds),
		TaskOK:    !rs.IsError,
	}, nil
}
