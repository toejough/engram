package cli

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"engram/internal/chat"
)

// acquireHoldSetup validates required flags and resolves the chat file path for hold acquire.
func acquireHoldSetup(holder, target, chatFileOverride string) (string, error) {
	if holder == "" {
		return "", errHolderRequired
	}

	if target == "" {
		return "", errTargetRequired
	}

	return resolveChatFile(chatFileOverride, "hold acquire", os.UserHomeDir, os.Getwd)
}

// filterHolds returns holds matching all non-empty filter criteria.
func filterHolds(holds []chat.HoldRecord, holder, target, tag string) []chat.HoldRecord {
	filtered := make([]chat.HoldRecord, 0, len(holds))

	for _, hold := range holds {
		if holder != "" && hold.Holder != holder {
			continue
		}

		if target != "" && hold.Target != target {
			continue
		}

		if tag != "" && hold.Tag != tag {
			continue
		}

		filtered = append(filtered, hold)
	}

	return filtered
}

// generateHoldID returns a UUID v4 string using crypto/rand.
func generateHoldID() (string, error) {
	var b [16]byte

	_, err := rand.Read(b[:])
	if err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}

	b[6] = (b[6] & uuidV4VersionBitmask) | uuidV4VersionByte
	b[8] = (b[8] & uuidV4VariantBitmask) | uuidV4VariantByte

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// marshalReleasePayload returns the JSON release payload for a hold ID.
// json.Marshal of a map[string]string is infallible; error is returned for interface conformance.
func marshalReleasePayload(holdID string) ([]byte, error) {
	return json.Marshal(map[string]string{"hold-id": holdID}) //nolint:wrapcheck
}

func runHoldAcquire(args []string, stdout io.Writer) error {
	fs := newFlagSet("hold acquire")

	holder := fs.String("holder", "", "agent acquiring the hold")
	target := fs.String("target", "", "agent being held")
	condition := fs.String("condition", "", "auto-release condition")
	tag := fs.String("tag", "", "workflow label for bulk operations (e.g. codesign-1)")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("hold acquire: %w", parseErr)
	}

	chatFilePath, acquireErr := acquireHoldSetup(*holder, *target, *chatFile)
	if acquireErr != nil {
		return acquireErr
	}

	holdID, genErr := generateHoldID()
	if genErr != nil {
		return fmt.Errorf("generating hold id: %w", genErr)
	}

	record := chat.HoldRecord{
		HoldID:     holdID,
		Holder:     *holder,
		Target:     *target,
		Condition:  *condition,
		Tag:        *tag,
		AcquiredTS: HoldNowFunc().UTC(),
	}

	text, marshalErr := json.Marshal(record)
	if marshalErr != nil {
		return fmt.Errorf("marshaling hold record: %w", marshalErr)
	}

	_, postErr := newFilePoster(chatFilePath).Post(chat.Message{
		From:   *holder,
		To:     *target,
		Thread: "hold",
		Type:   "hold-acquire",
		Text:   string(text),
	})
	if postErr != nil {
		return fmt.Errorf("hold acquire: posting: %w", postErr)
	}

	_, err := fmt.Fprintln(stdout, holdID)
	if err != nil {
		return fmt.Errorf("hold acquire: writing output: %w", err)
	}

	return nil
}

func runHoldCheck(args []string, stdout io.Writer) error {
	fs := newFlagSet("hold check")

	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("hold check: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "hold check", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	messages, loadErr := loadChatMessages(chatFilePath, os.ReadFile)
	if loadErr != nil {
		return fmt.Errorf("hold check: %w", loadErr)
	}

	activeHolds := chat.ScanActiveHolds(messages)
	poster := newFilePoster(chatFilePath)

	for _, hold := range activeHolds {
		met, _ := chat.EvaluateCondition(hold, messages)
		if !met {
			continue
		}

		releaseText, marshalErr := marshalReleasePayload(hold.HoldID)
		if marshalErr != nil {
			return fmt.Errorf("hold check: marshaling release: %w", marshalErr)
		}

		_, postErr := poster.Post(chat.Message{
			From:   "system",
			To:     "all",
			Thread: "hold",
			Type:   "hold-release",
			Text:   string(releaseText),
		})
		if postErr != nil {
			return fmt.Errorf("hold check: posting release for %s: %w", hold.HoldID, postErr)
		}

		_, writeErr := fmt.Fprintln(stdout, hold.HoldID)
		if writeErr != nil {
			return fmt.Errorf("hold check: writing output: %w", writeErr)
		}
	}

	return nil
}

// runHoldDispatch routes hold subcommands (acquire|release|list|check).
func runHoldDispatch(subArgs []string, stdout io.Writer) error {
	if len(subArgs) < 1 {
		return fmt.Errorf("%w: hold requires a subcommand (acquire|release|list|check)", errUsage)
	}

	switch subArgs[0] {
	case "acquire":
		return runHoldAcquire(subArgs[1:], stdout)
	case "release":
		return runHoldRelease(subArgs[1:], stdout)
	case "list":
		return runHoldList(subArgs[1:], stdout)
	case "check":
		return runHoldCheck(subArgs[1:], stdout)
	default:
		return fmt.Errorf("%w: hold %s", errUnknownCommand, subArgs[0])
	}
}

func runHoldList(args []string, stdout io.Writer) error {
	fs := newFlagSet("hold list")

	holder := fs.String("holder", "", "filter by holder agent name")
	target := fs.String("target", "", "filter by target agent name")
	tag := fs.String("tag", "", "filter by workflow tag")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("hold list: %w", parseErr)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "hold list", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	messages, loadErr := loadChatMessages(chatFilePath, os.ReadFile)
	if loadErr != nil {
		return fmt.Errorf("hold list: %w", loadErr)
	}

	enc := json.NewEncoder(stdout)
	for _, hold := range filterHolds(chat.ScanActiveHolds(messages), *holder, *target, *tag) {
		encErr := enc.Encode(hold)
		if encErr != nil {
			return fmt.Errorf("hold list: writing output: %w", encErr)
		}
	}

	return nil
}

func runHoldRelease(args []string, stdout io.Writer) error {
	fs := newFlagSet("hold release")

	holdID := fs.String("hold-id", "", "hold ID returned by engram hold acquire")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	parseErr := fs.Parse(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}

	if parseErr != nil {
		return fmt.Errorf("hold release: %w", parseErr)
	}

	if *holdID == "" {
		return fmt.Errorf("%w", errHoldIDRequired)
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "hold release", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	releasePayload, marshalErr := marshalReleasePayload(*holdID)
	if marshalErr != nil {
		return fmt.Errorf("hold release: marshaling: %w", marshalErr)
	}

	_, postErr := newFilePoster(chatFilePath).Post(chat.Message{
		From:   "system",
		To:     "all",
		Thread: "hold",
		Type:   "hold-release",
		Text:   string(releasePayload),
	})
	if postErr != nil {
		return fmt.Errorf("hold release: posting: %w", postErr)
	}

	_, err := fmt.Fprintln(stdout, "OK")
	if err != nil {
		return fmt.Errorf("hold release: writing output: %w", err)
	}

	return nil
}
