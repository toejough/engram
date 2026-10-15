package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/toejough/engram/internal/auto"
)

// unexported constants.
const (
	// the values of ENGRAM_AUTO_FAULT (793b418): the operations of the System's work fail, the lookups never answer, or
	// the work other than lookups takes the seconds that follow `delay:`
	autoFaultDelay = "delay:"
	// the fault seam is harness-only: the variables are read by NewDeps and no product path sets them
	autoFaultEnvVar    = "ENGRAM_AUTO_FAULT"
	autoFaultError     = "error"
	autoFaultHang      = "hang"
	autoFaultLogEnvVar = "ENGRAM_AUTO_FAULT_LOG"
	autoQueryOp        = "query"
	// the content fields of a fact and of a feedback note, by the name the engine and the exchange hash give them
	fieldAction    = "action"
	fieldBehavior  = "behavior"
	fieldImpact    = "impact"
	fieldObject    = "object"
	fieldPredicate = "predicate"
	fieldSubject   = "subject"
)

// unexported variables.
var (
	errAutoCrashed       = errors.New("crashed")
	errAutoCut           = errors.New("still running after")
	errAutoInjectedError = errors.New("engram: injected failure")
)

// autoVault is the vault of a job (793b417): the code of the commands query, show, learn, amend and ingest, called in
// the process with the dependencies the command line composes, each within the deadline of its step. The write lock
// of the vault and the lock of the manifest are taken by that code, as the commands take them.
type autoVault struct {
	deps Deps
	// ctx is the context of the `engram auto run` process, which every operation derives its deadline from
	ctx context.Context //nolint:containedctx // one process, one job: the context is the process's, not a request's
}

// Amend amends a note.
func (v autoVault) Amend(request auto.AmendRequest, deadline time.Duration) error {
	args := AmendArgs{
		Target: request.Target, Into: request.Into, ExpectHash: request.ExpectHash, Provenance: request.Provenance,
		Activate: request.Activate, ClearPending: request.ClearPending, Discard: request.Discard,
	}

	for _, field := range request.Fields {
		setAmendField(&args, field[0], field[1])
	}

	if request.ExpectNote != "" {
		hash, hashErr := exchangeHash([]byte(request.ExpectNote))
		if hashErr != nil {
			return fmt.Errorf("amend: the note as it was read: %w", hashErr)
		}

		args.ExpectHash = hash
	}

	_, err := v.operation("amend", deadline, func(ctx context.Context, out io.Writer) error {
		return amendCommand(logContext(v.deps)(ctx), v.deps, args, out)
	})

	return err
}

// Index brings the chunk index up to date.
func (v autoVault) Index(deadline time.Duration) error {
	_, err := v.operation("ingest", deadline, func(ctx context.Context, out io.Writer) error {
		return ingestCommand(logContext(v.deps)(ctx), v.deps, IngestArgs{Auto: true}, out)
	})

	return err
}

// Learn writes a note at the top of the vault and returns its path.
func (v autoVault) Learn(request auto.LearnRequest, deadline time.Duration) (string, error) {
	common := CommonLearnArgs{Slug: request.Slug, Position: "top", Source: request.Source}
	values := map[string]string{}

	for _, field := range request.Values {
		values[field[0]] = field[1]
	}

	return v.operation("learn", deadline, func(ctx context.Context, out io.Writer) error {
		ctx = logContext(v.deps)(ctx)

		if request.Kind == "fact" {
			return learnFactCommand(ctx, v.deps, LearnFactArgs{
				CommonLearnArgs: common, Situation: request.Situation,
				Subject: values[fieldSubject], Predicate: values[fieldPredicate], Object: values[fieldObject],
			}, out)
		}

		return learnFeedbackCommand(ctx, v.deps, LearnFeedbackArgs{
			CommonLearnArgs: common, Situation: request.Situation,
			Behavior: values[fieldBehavior], Impact: values[fieldImpact], Action: values[fieldAction],
		}, out)
	})
}

// Query searches the vault and the chunk index.
func (v autoVault) Query(request auto.QueryRequest, deadline time.Duration) (string, error) {
	args := QueryArgs{Phrases: request.Phrases, Text: request.Text, Limit: request.Limit}

	return v.operation(autoQueryOp, deadline, func(ctx context.Context, out io.Writer) error {
		return queryCommand(logContext(v.deps)(ctx), v.deps, args, out)
	})
}

// Show reads a note.
func (v autoVault) Show(name string, deadline time.Duration) (string, error) {
	return v.operation("show", deadline, func(ctx context.Context, out io.Writer) error {
		return showCommand(logContext(v.deps)(ctx), v.deps, ShowArgs{Ref: name}, out)
	})
}

// noteDelay appends the line the harness reads to know that slow work began: the second, and the operation. The line
// cannot fail the operation.
func (v autoVault) noteDelay(operation string) {
	file := v.deps.AutoFaultLog
	if file == "" {
		return
	}

	before, _ := v.deps.FS.ReadFile(file)
	line := fmt.Sprintf("%d slept-start %s\n", v.deps.Now().Unix(), operation)
	_ = v.deps.FS.WriteFile(file, append(before, line...), autoFilePerm)
}

// operation runs one operation within its deadline, and gives what it printed. An operation that passes its deadline
// is cut: its result is not waited for, and the error says it was still running. The fault of the harness, if one is
// set, stands before the operation.
func (v autoVault) operation(
	operation string, deadline time.Duration, command func(ctx context.Context, out io.Writer) error,
) (string, error) {
	limited, cancel := context.WithTimeout(v.ctx, deadline)
	defer cancel()

	var out bytes.Buffer

	finished := make(chan error, 1)

	go func() {
		// a failure of the code of a command is the failure of the operation, as the exit of a process was
		defer func() {
			if crashed := recover(); crashed != nil {
				finished <- fmt.Errorf("%w: %v", errAutoCrashed, crashed)
			}
		}()

		finished <- v.withFault(limited, operation, func() error { return command(limited, &out) })
	}()

	select {
	case err := <-finished:
		if err != nil {
			return "", fmt.Errorf("%s: %w", operation, err)
		}

		return out.String(), nil
	case <-limited.Done():
		return "", fmt.Errorf("%s: %w %dms", operation, errAutoCut, deadline.Milliseconds())
	}
}

// withFault performs the operation, after the fault of the harness (793b418): `error` fails every operation, `hang`
// lets a query run past every deadline, and `delay:N` makes every operation but a query wait N seconds, noting in the
// fault log that it began to.
func (v autoVault) withFault(ctx context.Context, operation string, perform func() error) error {
	fault := v.deps.AutoFault

	switch {
	case fault == autoFaultError:
		return errAutoInjectedError
	case fault == autoFaultHang && operation == autoQueryOp:
		<-ctx.Done()

		return fmt.Errorf("%w: %w", errAutoCut, ctx.Err())
	case strings.HasPrefix(fault, autoFaultDelay) && operation != autoQueryOp:
		seconds, _ := strconv.Atoi(strings.TrimPrefix(fault, autoFaultDelay))
		v.noteDelay(operation)

		select {
		case <-time.After(time.Duration(seconds) * time.Second):
		case <-ctx.Done():
			return fmt.Errorf("%w: %w", errAutoCut, ctx.Err())
		}
	}

	return perform()
}

// newAutoVault is the vault of the job `engram auto run` advances.
func newAutoVault(ctx context.Context, deps Deps) auto.Vault { return autoVault{deps: deps, ctx: ctx} }

// setAmendField sets the content field of an amend that the engine names.
func setAmendField(args *AmendArgs, name, value string) {
	fields := map[string]*string{
		fieldSubject: &args.Subject, fieldPredicate: &args.Predicate, fieldObject: &args.Object,
		fieldBehavior: &args.Behavior, fieldImpact: &args.Impact, fieldAction: &args.Action,
	}

	if target := fields[name]; target != nil {
		*target = value
	}
}
