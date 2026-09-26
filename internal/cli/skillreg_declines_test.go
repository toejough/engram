package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestReadSkillRegistrations_AcceptsVersion2 covers design D7: a v2 file
// reads like a v1 file.
func TestReadSkillRegistrations_AcceptsVersion2(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()
	vault.put("skill-registrations.json", `{"schema_version":2,"declined":{"superpowers:brainstorming":"h1"}}`)

	declined, err := cli.ReadSkillRegistrations("/vault", vault.readFile)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(declined).To(Equal(map[string]string{"superpowers:brainstorming": "h1"}))
}

// TestReadSkillRegistrations_ReadsVersion1AsIs covers "Version-1 decline
// still suppresses the offer": a v1 file's names are read as bare keys.
func TestReadSkillRegistrations_ReadsVersion1AsIs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()
	vault.put("skill-registrations.json", `{"schema_version":1,"declined":{"route":"h"}}`)

	declined, err := cli.ReadSkillRegistrations("/vault", vault.readFile)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(declined).To(Equal(map[string]string{"route": "h"}))
}

// TestReadSkillRegistrations_RejectsVersionAbove2 covers "Unknown schema
// version fails loudly": a version above 2 is an error, not an empty state.
func TestReadSkillRegistrations_RejectsVersionAbove2(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()
	vault.put("skill-registrations.json", `{"schema_version":3,"declined":{"route":"h"}}`)

	declined, err := cli.ReadSkillRegistrations("/vault", vault.readFile)

	g.Expect(err).To(MatchError(cli.ErrSkillRegistrationsVersionForTest))
	g.Expect(declined).To(BeNil())
}

// TestRecordSkillDeclined_FirstWriteStampsVersion2KeepingEveryEntry covers
// design D7's migration: the first write over a v1 file stamps v2 and keeps
// every v1 entry, in the unchanged {schema_version, declined} shape.
func TestRecordSkillDeclined_FirstWriteStampsVersion2KeepingEveryEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()
	vault.put("skill-registrations.json", `{"schema_version":1,"declined":{"route":"aaa","curate":"ccc"}}`)

	var written []byte

	err := cli.RecordSkillDeclined("/vault", "superpowers:brainstorming", "bbb", vault.readFile,
		func(_ string, data []byte) error {
			written = data

			return nil
		})

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(written)).To(Equal(
		`{"schema_version":2,"declined":{"curate":"ccc","route":"aaa","superpowers:brainstorming":"bbb"}}`))
}

// TestRecordSkillDeclined_OlderReaderDecodesVersion2 covers design D7's
// compatibility claim: the v2 file keeps exactly the fields the old reader
// knows, so a strict decode of that shape (unknown fields rejected, as
// ReadSkillRegistrations always did) still succeeds.
func TestRecordSkillDeclined_OlderReaderDecodesVersion2(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()

	var written []byte

	err := cli.RecordSkillDeclined("/vault", "pi:x", "h", vault.readFile, func(_ string, data []byte) error {
		written = data

		return nil
	})
	g.Expect(err).NotTo(HaveOccurred())

	var oldShape struct {
		SchemaVersion int               `json:"schema_version"` //nolint:tagliatelle // on-disk snake_case
		Declined      map[string]string `json:"declined"`
	}

	decoder := json.NewDecoder(bytes.NewReader(written))
	decoder.DisallowUnknownFields()

	g.Expect(decoder.Decode(&oldShape)).To(Succeed())
	g.Expect(oldShape.SchemaVersion).To(Equal(skillRegistrationsVersion2))
	g.Expect(oldShape.Declined).To(Equal(map[string]string{"pi:x": "h"}))
}

// TestRecordSkillDeclined_Property_KeepsEveryEntryAndStampsV2 is the
// invariant over any v1 or v2 decline state: a recorded decline yields a
// v2 file holding every prior entry plus the new one (overwriting only its
// own key), and reading it back returns exactly that map.
func TestRecordSkillDeclined_Property_KeepsEveryEntryAndStampsV2(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		keyGen := rapid.StringMatching(`[a-z]{1,3}(:[a-z]{1,3}){0,2}`)
		prior := rapid.MapOf(keyGen, rapid.StringMatching(`[0-9a-f]{1,6}`)).Draw(rt, "prior")
		version := rapid.IntRange(1, skillRegistrationsVersion2).Draw(rt, "version")
		key := keyGen.Draw(rt, "key")
		hash := rapid.StringMatching(`[0-9a-f]{1,6}`).Draw(rt, "hash")

		priorJSON, marshalErr := json.Marshal(prior)
		g.Expect(marshalErr).NotTo(HaveOccurred())

		vault := newSkillregFixtureVault()
		vault.put("skill-registrations.json",
			fmt.Sprintf(`{"schema_version":%d,"declined":%s}`, version, priorJSON))

		err := cli.RecordSkillDeclined("/vault", key, hash, vault.readFile, func(path string, data []byte) error {
			vault.files[path] = string(data)

			return nil
		})
		g.Expect(err).NotTo(HaveOccurred())

		want := maps.Clone(prior)
		want[key] = hash

		var doc struct {
			SchemaVersion int               `json:"schema_version"` //nolint:tagliatelle // on-disk snake_case
			Declined      map[string]string `json:"declined"`
		}

		g.Expect(json.Unmarshal([]byte(vault.files["/vault/skill-registrations.json"]), &doc)).To(Succeed())
		g.Expect(doc.SchemaVersion).To(Equal(skillRegistrationsVersion2))
		g.Expect(doc.Declined).To(Equal(want))

		reread, readErr := cli.ReadSkillRegistrations("/vault", vault.readFile)
		g.Expect(readErr).NotTo(HaveOccurred())
		g.Expect(reread).To(Equal(want))
	})
}

// TestRecordSkillDeclined_VersionAbove2WritesNothing covers "Unknown schema
// version fails loudly ... writes nothing".
func TestRecordSkillDeclined_VersionAbove2WritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()
	vault.put("skill-registrations.json", `{"schema_version":3,"declined":{}}`)

	err := cli.RecordSkillDeclined("/vault", "curate", "h", vault.readFile, func(string, []byte) error {
		g.Fail("a version-3 file must never be overwritten")

		return nil
	})

	g.Expect(err).To(MatchError(cli.ErrSkillRegistrationsVersionForTest))
}

// TestRunSkillRegistration_VersionAbove2_ErrorsAndWritesNothing covers the
// scenario end to end: registration reports the error and writes nothing,
// even with an explicit --accept.
func TestRunSkillRegistration_VersionAbove2_ErrorsAndWritesNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillAcceptFixtureVault()
	vault.put("skill-registrations.json", `{"schema_version":3,"declined":{}}`)

	sourceFS := skillsHomeFixture(map[string][]byte{"curate": []byte("# Curate\n")})
	deps := skillRegistrationDepsFor(vault, sourceFS)

	var stdout bytes.Buffer

	err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
		Vault: "/vault", VaultName: "personal", Home: skillRegHome, Accept: []string{"curate"},
	}, deps, &stdout)

	g.Expect(err).To(MatchError(cli.ErrSkillRegistrationsVersionForTest))
	g.Expect(vault.files).To(HaveLen(1))
}

// unexported constants.
const (
	// skillRegistrationsVersion2 is design D7's current decline-file schema.
	skillRegistrationsVersion2 = 2
)
