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

// TestReadSkillRegistrations_AcceptsVersion2 covers design D7: a v2 file's
// declines are read keyed by skill key.
func TestReadSkillRegistrations_AcceptsVersion2(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newSkillregFixtureVault()
	vault.put("skill-registrations.json", `{"schema_version":2,"declined":{"superpowers:brainstorming":"h1"}}`)

	declined, err := cli.ReadSkillRegistrations("/vault", vault.readFile)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(declined).To(Equal(map[string]string{"superpowers:brainstorming": "h1"}))
}

// TestReadSkillRegistrations_RejectsEveryVersionButTwo covers "Version-1 file
// fails loudly" and "Unknown schema version fails loudly" (design D7: v1
// reading is dropped): a schema_version below 2, above 2 or missing is an
// error, never an empty decline state.
func TestReadSkillRegistrations_RejectsEveryVersionButTwo(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"version 1": `{"schema_version":1,"declined":{"route":"h"}}`,
		"version 3": `{"schema_version":3,"declined":{"route":"h"}}`,
		"missing":   `{"declined":{"route":"h"}}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillregFixtureVault()
			vault.put("skill-registrations.json", body)

			declined, err := cli.ReadSkillRegistrations("/vault", vault.readFile)

			g.Expect(err).To(MatchError(cli.ErrSkillRegistrationsVersionForTest))
			g.Expect(declined).To(BeNil())
		})
	}
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
// invariant over any v2 decline state: a recorded decline yields a
// v2 file holding every prior entry plus the new one (overwriting only its
// own key), and reading it back returns exactly that map.
func TestRecordSkillDeclined_Property_KeepsEveryEntryAndStampsV2(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		keyGen := rapid.StringMatching(`[a-z]{1,3}(:[a-z]{1,3}){0,2}`)
		prior := rapid.MapOf(keyGen, rapid.StringMatching(`[0-9a-f]{1,6}`)).Draw(rt, "prior")
		key := keyGen.Draw(rt, "key")
		hash := rapid.StringMatching(`[0-9a-f]{1,6}`).Draw(rt, "hash")

		priorJSON, marshalErr := json.Marshal(prior)
		g.Expect(marshalErr).NotTo(HaveOccurred())

		vault := newSkillregFixtureVault()
		vault.put("skill-registrations.json",
			fmt.Sprintf(`{"schema_version":%d,"declined":%s}`, skillRegistrationsVersion2, priorJSON))

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

// TestRunSkillRegistration_NonVersion2File_ErrorsAndWritesNothing covers
// "Version-1 file fails loudly" and "Unknown schema version fails loudly" end
// to end: registration reports the error and writes nothing, even with an
// explicit --accept.
func TestRunSkillRegistration_NonVersion2File_ErrorsAndWritesNothing(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"version 1": `{"schema_version":1,"declined":{"route":"h"}}`,
		"version 3": `{"schema_version":3,"declined":{}}`,
		"missing":   `{"declined":{}}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newSkillAcceptFixtureVault()
			vault.put("skill-registrations.json", body)

			sourceFS := skillsHomeFixture(map[string][]byte{"curate": []byte("# Curate\n")})
			deps := skillRegistrationDepsFor(vault, sourceFS)

			var stdout bytes.Buffer

			err := cli.RunSkillRegistration(t.Context(), cli.SkillRegistrationArgs{
				Vault: "/vault", VaultName: "personal", Home: skillRegHome, Accept: []string{"claude:curate"},
			}, deps, &stdout)

			g.Expect(err).To(MatchError(cli.ErrSkillRegistrationsVersionForTest))
			g.Expect(vault.files).To(Equal(map[string]string{"/vault/skill-registrations.json": body}))
		})
	}
}

// TestSkillRegistrations_Property_OnlyVersion2IsRead is the invariant over
// any schema_version, present or missing: version 2 reads back its declines;
// every other version (missing included) is errSkillRegistrationsVersion on
// read, and RecordSkillDeclined over it errors without writing.
func TestSkillRegistrations_Property_OnlyVersion2IsRead(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		declinedJSON := `{"claude:route":"h"}`
		body := `{"declined":` + declinedJSON + `}`
		version, present := 0, rapid.Bool().Draw(rt, "present")

		if present {
			version = rapid.IntRange(-3, 9).Draw(rt, "version")
			body = fmt.Sprintf(`{"schema_version":%d,"declined":%s}`, version, declinedJSON)
		}

		vault := newSkillregFixtureVault()
		vault.put("skill-registrations.json", body)

		declined, readErr := cli.ReadSkillRegistrations("/vault", vault.readFile)

		wrote := false
		recordErr := cli.RecordSkillDeclined("/vault", "claude:curate", "h2", vault.readFile,
			func(string, []byte) error {
				wrote = true

				return nil
			})

		if present && version == skillRegistrationsVersion2 {
			g.Expect(readErr).NotTo(HaveOccurred())
			g.Expect(declined).To(Equal(map[string]string{"claude:route": "h"}))
			g.Expect(recordErr).NotTo(HaveOccurred())
			g.Expect(wrote).To(BeTrue())

			return
		}

		g.Expect(readErr).To(MatchError(cli.ErrSkillRegistrationsVersionForTest))
		g.Expect(declined).To(BeNil())
		g.Expect(recordErr).To(MatchError(cli.ErrSkillRegistrationsVersionForTest))
		g.Expect(wrote).To(BeFalse(), "a non-v2 file must never be overwritten")
	})
}

// unexported constants.
const (
	// skillRegistrationsVersion2 is design D7's current decline-file schema.
	skillRegistrationsVersion2 = 2
)
