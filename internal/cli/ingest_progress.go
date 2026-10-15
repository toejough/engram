package cli

import (
	"slices"
	"strings"
	"time"
)

// unexported constants.
const (
	// autoLockWait is how long an --auto ingest waits for the manifest lock before it skips: long enough for the run
	// of another session that is only checking an index that is up to date, short enough that a run that is catching
	// the index up (minutes) does not hold the turn's other work (793d39c1).
	autoLockWait = 5 * time.Second
	// manifestCheckpointInterval is the most often a run writes the manifest while it works: the manifest of a real
	// corpus is tens of megabytes, so a write per source would cost more than the source (793d39b1).
	manifestCheckpointInterval = 10 * time.Second
)

// manifestSaver writes the manifest as a run progresses, at the boundary of a group of identical copies, so that a
// run that is cut by its deadline or killed has advanced the index by the sources it completed (793d39b1). Writes are
// at most one per manifestCheckpointInterval of the injected clock; with no clock only flush writes.
type manifestSaver struct {
	chunksDir string
	manifest  ingestManifest
	deps      IngestDeps
	lastWrite time.Time
	dirty     bool
}

// checkpoint records that a group changed the manifest, and writes it when the interval has passed.
func (s *manifestSaver) checkpoint(changed bool) error {
	s.dirty = s.dirty || changed

	if !s.dirty || s.deps.Now == nil || s.now().Sub(s.lastWrite) < manifestCheckpointInterval {
		return nil
	}

	return s.flush()
}

// flush writes the manifest when it has changed since it was last written.
func (s *manifestSaver) flush() error {
	if !s.dirty {
		return nil
	}

	err := writeManifestFile(s.chunksDir, s.manifest, s.deps)
	if err != nil {
		return err
	}

	s.dirty, s.lastWrite = false, s.now()

	return nil
}

func (s *manifestSaver) now() time.Time {
	if s.deps.Now == nil {
		return time.Time{}
	}

	return s.deps.Now()
}

// acquireIngestLock takes the manifest lock for a run. A named ingest waits for it, as it always has. An --auto
// ingest waits at most its lock wait and then skips with ErrIngestBusy, so that two sessions never index the same
// backlog twice and a turn does not wait a deadline for another run (793d39c1). A grant that arrives after the skip is
// given back at once.
func acquireIngestLock(auto bool, chunksDir string, deps IngestDeps) (func(), error) {
	if !auto || deps.Lock == nil {
		return acquireOptionalLock(deps.Lock, chunksDir)
	}

	type grant struct {
		release func()
		err     error
	}

	granted := make(chan grant, 1)

	go func() {
		release, err := deps.Lock(chunksDir)
		granted <- grant{release: release, err: err}
	}()

	wait := deps.LockWait
	if wait <= 0 {
		wait = autoLockWait
	}

	timer := time.NewTimer(wait)

	defer timer.Stop()

	select {
	case result := <-granted:
		return result.release, result.err
	case <-timer.C:
		go func() {
			if late := <-granted; late.err == nil {
				late.release()
			}
		}()

		return nil, ErrIngestBusy
	}
}

func newManifestSaver(chunksDir string, manifest ingestManifest, deps IngestDeps) *manifestSaver {
	saver := &manifestSaver{chunksDir: chunksDir, manifest: manifest, deps: deps}
	saver.lastWrite = saver.now()

	return saver
}

// orderedGroups lists the hash groups in the order a run works them: the groups with a source that must be read
// (new or changed) first, the most recently modified first, so that a run that is cut has indexed the sessions the
// backlog captures first; then the groups with nothing to do, which cost a comparison each. Ties break on the
// lowest path, so the order is the same on every run (793d39b1).
func orderedGroups(hashed []hashedSource, deps IngestDeps) [][]hashedSource {
	type ranked struct {
		members []hashedSource
		work    bool
		newest  int64
		first   string
	}

	grouped := groupByHash(hashed)
	groups := make([]ranked, 0, len(grouped))

	for _, members := range grouped {
		group := ranked{members: members, first: members[0].ref.path}

		for _, member := range members {
			group.first = min(group.first, member.ref.path)

			if member.raw != nil {
				group.work = true
				group.newest = max(group.newest, statOrZero(deps, member.ref.path).MtimeUnixNano)
			}
		}

		groups = append(groups, group)
	}

	slices.SortFunc(groups, func(a, b ranked) int {
		if a.work != b.work {
			if a.work {
				return -1
			}

			return 1
		}

		switch {
		case a.newest > b.newest:
			return -1
		case a.newest < b.newest:
			return 1
		}

		return strings.Compare(a.first, b.first)
	})

	ordered := make([][]hashedSource, len(groups))
	for i, group := range groups {
		ordered[i] = group.members
	}

	return ordered
}
