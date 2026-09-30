package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/reviewtransaction"
)

func TestReviewProviderArtifactV1ContractsArePinned(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration", "v1")
	want := map[string]string{
		// Ordo: command renamed to ordo.
		"fixtures/capabilities-v1.4.fixture.json": "7c7cd6670ede933f132a14c973288799370e8ddf4abecddb784db096956a5e29",
		// Ordo: the consent headline names the active brand (default "Ordo").
		// Ordo: command renamed to ordo.
		"fixtures/consent.fixture.json": "b7b6fde79232acee74580e182ee7d13214870944b8571a43f6eb1039c0c8dfd8",
		"fixtures/start.fixture.json":   "3b963b221cd1560eb8872cbabbb5407096f593ced2f13eb9cb06eb61e4cca4d1",
		// issue #2659: start-v2/status-v2 embed a freshly minted target_identity,
		// and the purified identity domain legitimately changed that hash for
		// every new snapshot. Deliberate, not drift.
		"fixtures/start-v2.fixture.json":         "2699660832c0d944184d5d314f08774ab9a02f5b8a7a4c2a07983440e0e346ad",
		"fixtures/status.fixture.json":           "a1f28b7d5351e000aca5238ed6348a0838fe2c0e64ce894ebcd8b43851063ff6",
		"fixtures/status-v2.fixture.json":        "ff3690a9e716c9fa48e3c26a67047f9b4ce4c3cce8391a240dbe9834bd4e13ee",
		"fixtures/status-ambiguous.fixture.json": "ee695fd58ba72adfb3b51dfd16432a177498173a45bfcb594d6bdc53bfa32e6e",
		"fixtures/status-corrupted.fixture.json": "4cfc0048c28a39cec8a32fecfaad66e56e5c1248263ceb4ce66b6717981880b2",
		"fixtures/status-recover.fixture.json":   "714f762f72380ce93d567626cafbaa536ab3aae02af73d3d40ca123f1f30d8b0",
		"fixtures/status-unrelated.fixture.json": "deab36c877ced3c9b480ca33724c10d88f75c761d6426fa14be850345122891d",
		"schemas/admitted-result.schema.json":    "7796e8dbba331434594108c902dfab7ec46f691fa447a9259a78f2448111b0de",
		"schemas/artifact-subject.schema.json":   "f7dcd934e27e8f3735a37f3d0ec8048dd8ccc1811b9df61124a1dcbf8a03f40e",
		// Ordo: command renamed to ordo.
		"schemas/capabilities-v1.4.schema.json": "7d2f3237bebd36f8e7cfc6e4b58b75825d065db05b98b349b5617812ddd38d1c",
		"schemas/result-artifact.schema.json":   "91296bd2c261fd2fe03bffd63efe58badd4927e0d0d8480cd4213f651ecacdf6",
		"schemas/start.schema.json":             "4296aebbd4128ce51945a2f6d3228aa77ac7215c802978d559bff5279ec56229",
		// Frozen v1 START artifacts do not project the v3 replay or retired
		// stale-burn fields.
		"schemas/start-v2.schema.json": "ec8550cd93bbe84af1ce87dfd7abfa9e24692f42b20f8f0bf9cac1d4b88ea46c",
		// Ordo: command renamed to ordo.
		"schemas/status.schema.json":    "b2c1f462acbe3abe6f552b8ad604902ebf42c762f5848759ae157ae858724977",
		"schemas/status-v2.schema.json": "7c51627d133592839ba4afa860b358b68109afd5f70ee998cd421f563201b23e",
		// Ordo: command renamed to ordo.
		"schemas/transition-execution.schema.json": "20eaa70a337c5a614c4a96f6d524b857a6534808cbe2b9ae06bc106516f607f4",
	}
	for name, expected := range want {
		payload, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(payload)
		if actual := hex.EncodeToString(digest[:]); actual != expected {
			t.Fatalf("%s digest = %s, want %s", name, actual, expected)
		}
	}
}

func TestReviewProviderArtifactV20ContractsArePinned(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration", "v2")
	want := map[string]string{
		// Ordo: command renamed to ordo.
		"fixtures/capabilities.fixture.json": "4bca670e55a7911dec415205037a051dab5bf7477bc13e20cd95bd0297550d86",
		// Ordo: the consent headline names the active brand (default "Ordo").
		// Ordo: command renamed to ordo.
		"fixtures/consent.fixture.json": "978c847365090a946f4a56392e966c8b19203b1df9a9815f0fda80ef0bfd42bf",
		// issue #3922 / #4199 / gentle-pi#543: the native-git reviewer_result
		// collect input no longer inlines changed_path_manifest -- it is
		// already committed to by artifact_subject.changed_path_manifest_sha256
		// -- so this fixture legitimately dropped that array. Deliberate, not
		// drift.
		"fixtures/status.fixture.json": "3fc2539d5bcaa8dc3ed650ba7f5e8915856a3d9f8caf1cfcb1b0354ecacbe0f8",
		// Ordo: command renamed to ordo.
		"schemas/capabilities.schema.json": "6bcf9145e17ab90dac5f8f25bd596ea3f8d30d278623ca785fa8a9a8d467968b",
		// Ordo: command renamed to ordo.
		"schemas/consent.schema.json": "a9b2179c968f2361199aabcc80bb5b3c087d03aa074c675714294a65b875055c",
		"schemas/status.schema.json":  "3b257b417270744061dc943a97537e253e36e34de4591b0400e3c38ea3efde80",
	}
	for name, expected := range want {
		payload, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(payload)
		if actual := hex.EncodeToString(digest[:]); actual != expected {
			t.Fatalf("%s digest = %s, want %s", name, actual, expected)
		}
	}
}

func TestReviewProviderArtifactV21ContractsArePinned(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration", "v2")
	want := map[string]string{
		// Ordo: command renamed to ordo.
		"fixtures/capabilities-v2.1.fixture.json": "2e21f6348ebe069b80a09f9396b7bea7f99b719ae562d80cf6a074687b650e72",
		// issue #2659: consent-v3 embeds a freshly minted target_identity;
		// the purified identity domain legitimately changed that hash.
		// Deliberate, not drift.
		// Ordo: the consent headline names the active brand (default "Ordo").
		// Ordo: command renamed to ordo.
		"fixtures/consent-v3.fixture.json": "21c96bb8c7cca9032f6cfdae951555670cac38425fdc8d56993a1d72c25ee5cb",
		// Ordo: command renamed to ordo.
		"schemas/capabilities-v2.1.schema.json": "2efbf922517e7130d78c0b27eeab0f6015ad5b1b8c8d83fdafb36f26c1e469a2",
		// Cross-lane battery conformance fix: the schema pinned the choice
		// invocations to `--agent claude-code`, but the live emitter omits the
		// agent token when the caller declared no runtime (the pinned fixture
		// itself carries no --agent), and #2676 binds the declared runtime
		// (claude-code, opencode, codex) when there is one. The schema now
		// follows the emitter. The agent enum also admits "pi": the Pi host
		// relay drives consent with its own declared runtime identity, which
		// the emitter legitimately publishes once the relay handshake is
		// declared. Deliberate, not drift.
		// Ordo: command renamed to ordo.
		"schemas/consent-v3.schema.json": "bf5ef93a0e5371f2b389669893627abd60d014d1f8341de5c03de507c83096d1",
		"schemas/status.schema.json":     "3b257b417270744061dc943a97537e253e36e34de4591b0400e3c38ea3efde80",
	}
	for name, expected := range want {
		payload, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(payload)
		if actual := hex.EncodeToString(digest[:]); actual != expected {
			t.Fatalf("%s digest = %s, want %s", name, actual, expected)
		}
	}
}

func TestReviewProviderArtifactV25StatusContractsArePinned(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration", "v2")
	want := map[string]string{
		"fixtures/status-v5.fixture.json": "da401836833192a400493787b256b5f19b3a5ec5fd325ad45d8dcaeadfeea81e",
		// Cross-lane battery conformance fix: live negotiated STATUS publishes
		// the top-level repository_context reference (review_status_contract.go's
		// ReviewTargetStatusResult, populated since the recovered-units merges),
		// which the schema did not admit; and targeted_validation_required also
		// arrives as the provider-task (external.run_provider_role) and pi
		// host-relay (review.capture-validation) shapes, which the schema
		// rejected as missing the generic submission; and the negotiated-route
		// disposition preview (ReviewRepairDispositionProviderInputs) is real
		// optional emitter output the strict schema must admit; and the pi
		// host-relay materialize path renders the reviewer_result collect
		// input with a capture-result submission descriptor, which the
		// submission oneOf and the no-submission allOf rule both rejected.
		// Deliberate, not drift.
		//
		// issue #3922 / #4199 / gentle-pi#543: the review.capture-result `then`
		// clause no longer requires changed_path_manifest -- it stays an
		// allowed property, but the native-git transport no longer needs to
		// inline it since artifact_subject.changed_path_manifest_sha256 already
		// commits to it. Deliberate, not drift.
		"schemas/start.schema.json":     "27954ad34319719a68f90768c90f39254d94c62cf7f8ea90525ec4e2dbafd182",
		"schemas/status-v5.schema.json": "8f6d05bd4ed64abc765bd7ce9ae8bed0470448cd260fc0a94dc5929b88f42a18",
	}
	for name, expected := range want {
		payload, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(payload)
		if actual := hex.EncodeToString(digest[:]); actual != expected {
			t.Fatalf("%s digest = %s, want %s", name, actual, expected)
		}
	}
}

// TestReviewProviderArtifactV23StartContractsArePinned pins the artifacts the
// start/v4 continuation work first published (issue #3894): the start-v4
// envelope with its provider-issued reviewing STATUS re-entry, and the v2.3
// capabilities advertisement that names it.
func TestReviewProviderArtifactV23StartContractsArePinned(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration", "v2")
	want := map[string]string{
		// Ordo: command renamed to ordo.
		"fixtures/capabilities-v2.3.fixture.json": "1a1d6834c79b7ac4c0fbdc6ebf97e1973e3ee8651a3847834c2c235f15114d68",
		// Ordo: command renamed to ordo.
		"fixtures/start-v4.fixture.json": "6d1ec9dea69ff8bf584cd9c533adb35724dd9da9d1d04bcc78f39a279ae65a28",
		// Ordo: command renamed to ordo.
		"schemas/capabilities-v2.3.schema.json": "5d2a3d695ee48b10679611f23bcd135f6bc378daf1677d70bde9122b0d982655",
		"schemas/start-v4.schema.json":          "770c6a7e40a62a945d1134cba933cfd811f4c5e6ab407a36a26ba56508bc00e4",
	}
	for name, expected := range want {
		payload, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(payload)
		if actual := hex.EncodeToString(digest[:]); actual != expected {
			t.Fatalf("%s digest = %s, want %s", name, actual, expected)
		}
	}
}

func TestReviewProviderArtifactV24IntendedUntrackedContractsArePinned(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration", "v2")
	want := map[string]string{
		"schemas/capabilities-v2.4.schema.json":            "fc4d55dbad6b19cc4c289e8ed94bd1839800ca2892e449640459b668e0c7b0b5",
		"schemas/intended-untracked-selection.schema.json": "6f300c4cc10ab669fa3ef8cc608829df623a453cd5e6629958786e0724430259",
		"schemas/status-v6.schema.json":                    "0aa731e4d3961d678b4e51a6be0af93f2de82a4a326c3366e2fbe6a3e687236c",
	}
	for name, expected := range want {
		payload, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(payload)
		if actual := hex.EncodeToString(digest[:]); actual != expected {
			t.Fatalf("%s digest = %s, want %s", name, actual, expected)
		}
	}
}

// TestReviewProviderArtifactConformanceSchemasArePinned pins the schemas the
// cross-lane battery conformance work first published: the delivery gate
// result (gentle-ai.review-gate-result/v1) and the OpenCode provider-role
// capture acknowledgement (gentle-ai.opencode-review-provider-role/v1). Both
// envelopes already shipped on the wire; only their published schemas are new.
func TestReviewProviderArtifactConformanceSchemasArePinned(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration", "v2")
	want := map[string]string{
		"schemas/gate-result.schema.json": "afe5e2a030fae9949305811bcac0a6dbc8b4f28802fa61d1e31e58e895f9fcae",
		// issues #4226/#4453: last-event-closure documents terminal escalation
		// and exposes complete admitted reviewer results before acknowledgement.
		// Deliberate, not drift.
		"schemas/last-event-closure.schema.json": "08a94def144d70c0c1e84b73a1a245cd36bc73dc1ade4aa733b42097b05593eb",
		// issue #3894: start/v4 publishes the reviewing status continuation, so
		// transition-execution gains the start_status_execution definition it
		// references. Deliberate, not drift.
		// issue #3932: start_status_execution carries the opaque
		// repository-context row, so a foreign process cwd fails closed.
		// Ordo: command renamed to ordo.
		"schemas/transition-execution.schema.json":   "84073cc88c725837466b4011a93c8fe3d30c2184970ab86414f24e6fd4f4c7af",
		"schemas/opencode-provider-role.schema.json": "c6b9f216f89c044f8e844b55e7200114850cfbc16642bca0677f30a399d8aa9b",
	}
	for name, expected := range want {
		payload, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(payload)
		if actual := hex.EncodeToString(digest[:]); actual != expected {
			t.Fatalf("%s digest = %s, want %s", name, actual, expected)
		}
	}
}

// TestReviewProviderArtifactStatusV7ContractsArePinned pins the schemas issue
// #4040's fix first published: status/v7, which publishes
// eligible_untracked_inventory unconditionally at the top level, and
// capabilities-v2.5, which advertises status/v7 (design decision 5).
func TestReviewProviderArtifactStatusV7ContractsArePinned(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration", "v2")
	want := map[string]string{
		// issue #3928: the root action enum gained "collect" and "execute", the
		// v7 envelope's projection of a live transaction whose next_transition
		// is mandatory. Deliberate, not drift.
		//
		// issues #3299, #4170: a stale managed-asset digest now fails STATUS's
		// own preflight, before a START is ever offered, as a typed
		// managed_assets_outdated "stop" that carries the exact
		// candidate-preserving `ordo sync` continuation (see
		// failure.schema.json#/$defs/managed_assets_continuation). Deliberate,
		// not drift.
		// issue #3442: next_transition gained a third oneOf branch for the
		// unachievable_lens_slot stop, carrying one unachievable_lens_slots
		// entry per declared slot (lens, subject_hash, reason, and the exact
		// review.capture-unachievable --withdraw=true command), so a
		// restarted orchestrator that lost the pre-stop collect offer can
		// still recover the binding its withdraw needs. Deliberate, not drift.
		//
		// issue #4226: status/v7 documents the CompactEscalationEvidence contract
		// on terminal escalation. Deliberate, not drift.
		// Ordo: command renamed to ordo.
		"schemas/status-v7.schema.json":         "f1e8625dd588b672120c3430b49de3676023ed2a6bbe481eb81e26c445d09f73",
		"schemas/capabilities-v2.5.schema.json": "9fcdb1717a54bcd4f73d4dee1283d9ec2f27cccbb5d54804ee8b40a6ed2db553",
	}
	for name, expected := range want {
		payload, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(payload)
		if actual := hex.EncodeToString(digest[:]); actual != expected {
			t.Fatalf("%s digest = %s, want %s", name, actual, expected)
		}
	}
}

// TestManagedAssetsContinuationSchemaRuleIsExercised is the RED-first proof
// for the corroborated review finding on #3299/#4170: failure.schema.json's
// if/then rule tying `continuation` to `code: managed_assets_outdated` (and
// forbidding it on every other code), and status-v9.schema.json's dedicated
// oneOf branch for the same reason code, were published without ever
// validating a real produced envelope against either rule in either
// direction. It exercises both published schemas against three shapes each:
// the real envelope a stale managed-asset digest produces (must pass), the
// same envelope with `continuation` stripped (must fail the rule), and the
// same envelope with `continuation` attached to an unrelated code/reason
// (must fail the rule).
func TestManagedAssetsContinuationSchemaRuleIsExercised(t *testing.T) {
	home, repo := reviewEnabledHome(t), initReviewCLIRepo(t)
	writeReviewStartCandidate(t, repo, "docs/schema-rule.md", "# Candidate\n", 0o644)
	staleManagedReviewerAssets(t, home)

	// --- failure.schema.json, exercised against the real START preflight failure ---
	var startOutput bytes.Buffer
	if err := RunReview(boundNegotiatedStartArgs(t, []string{
		"start", "--contract", ReviewIntegrationContractV2, "--cwd", repo, "--agent", "opencode", "--consent", "granted",
	}), &startOutput); err == nil {
		t.Fatalf("stale managed assets START succeeded: %s", startOutput.String())
	}
	failureSchema := compileWholePublishedReviewSchema(t, "v2", "failure.schema.json")

	var failureDoc map[string]any
	if err := json.Unmarshal(startOutput.Bytes(), &failureDoc); err != nil {
		t.Fatal(err)
	}
	if failureDoc["code"] != "managed_assets_outdated" || failureDoc["continuation"] == nil {
		t.Fatalf("baseline stale managed assets failure = %#v", failureDoc)
	}

	// (a) the real envelope validates.
	validatePublishedReviewSchema(t, failureSchema, startOutput.Bytes())

	// (b) the same envelope with continuation removed must fail: the code
	// still claims managed_assets_outdated, and the rule requires it.
	withoutContinuation := decodeJSONObjectCopy(t, startOutput.Bytes())
	delete(withoutContinuation, "continuation")
	if err := failureSchema.Validate(withoutContinuation); err == nil {
		t.Fatal("failure.schema.json accepted a managed_assets_outdated code with no continuation")
	}

	// (c) the same continuation attached to an unrelated code must fail: the
	// rule's "else" branch forbids continuation on every other code.
	unrelatedCode := decodeJSONObjectCopy(t, startOutput.Bytes())
	unrelatedCode["code"] = "invalid_request"
	if err := failureSchema.Validate(unrelatedCode); err == nil {
		t.Fatal("failure.schema.json accepted a continuation on an unrelated failure code")
	}
	for _, command := range []string{
		"'/tmp/gentle\nai' sync --agent opencode",
		"\"C:\\\\gentle\r-ai.exe\" sync --agent opencode",
	} {
		multilineCommand := decodeJSONObjectCopy(t, startOutput.Bytes())
		multilineContinuation := multilineCommand["continuation"].(map[string]any)
		multilineContinuation["command"] = command
		if err := failureSchema.Validate(multilineCommand); err == nil {
			t.Fatalf("failure.schema.json accepted multiline continuation command %q", command)
		}
	}

	// --- status-v9.schema.json, exercised against the real STATUS stop transition ---
	var stopOutput bytes.Buffer
	if err := RunReview([]string{
		"status", "--cwd", repo, "--contract", ReviewIntegrationContractV2, "--agent", "opencode", "--next-transition",
	}, &stopOutput); err != nil {
		t.Fatalf("stale managed assets STATUS: %v\n%s", err, stopOutput.String())
	}
	statusV9Schema := compileWholeNativeStatusSchema(t, "status-v9.schema.json")

	statusDoc := decodeJSONObjectCopy(t, stopOutput.Bytes())
	nextTransition, ok := statusDoc["next_transition"].(map[string]any)
	if !ok || nextTransition["kind"] != "stop" || nextTransition["reason_code"] != "managed_assets_outdated" || nextTransition["continuation"] == nil {
		t.Fatalf("baseline stale managed assets STATUS next_transition = %#v", statusDoc["next_transition"])
	}

	// (a) the real envelope validates.
	validatePublishedReviewSchema(t, statusV9Schema, stopOutput.Bytes())

	// (b) the same stop with continuation removed must fail: nothing else in
	// the oneOf admits a managed_assets_outdated stop without it.
	withoutStopContinuation := decodeJSONObjectCopy(t, stopOutput.Bytes())
	withoutTransition := decodeJSONObjectCopy(t, stopOutput.Bytes())["next_transition"].(map[string]any)
	delete(withoutTransition, "continuation")
	withoutStopContinuation["next_transition"] = withoutTransition
	if err := statusV9Schema.Validate(withoutStopContinuation); err == nil {
		t.Fatal("status-v9.schema.json accepted a managed_assets_outdated stop with no continuation")
	}

	// (c) the same continuation attached to an unrelated reason code (still a
	// stop, e.g. rdd_disabled) must fail both oneOf branches: the generic one
	// forbids the extra continuation property, and the dedicated one requires
	// reason_code const managed_assets_outdated.
	unrelatedReason := decodeJSONObjectCopy(t, stopOutput.Bytes())
	unrelatedTransition := decodeJSONObjectCopy(t, stopOutput.Bytes())["next_transition"].(map[string]any)
	unrelatedTransition["reason_code"] = "rdd_disabled"
	unrelatedReason["next_transition"] = unrelatedTransition
	if err := statusV9Schema.Validate(unrelatedReason); err == nil {
		t.Fatal("status-v9.schema.json accepted a continuation attached to an unrelated stop reason code")
	}
	for _, command := range []string{
		"'/tmp/gentle\nai' sync --agent opencode",
		"\"C:\\\\gentle\r-ai.exe\" sync --agent opencode",
	} {
		multilineCommand := decodeJSONObjectCopy(t, stopOutput.Bytes())
		multilineTransition := multilineCommand["next_transition"].(map[string]any)
		multilineContinuation := multilineTransition["continuation"].(map[string]any)
		multilineContinuation["command"] = command
		if err := statusV9Schema.Validate(multilineCommand); err == nil {
			t.Fatalf("status-v9.schema.json accepted multiline continuation command %q", command)
		}
	}
}

// decodeJSONObjectCopy decodes payload into a fresh map[string]any, so a
// caller can mutate one field without aliasing any other test's copy of the
// same bytes.
func decodeJSONObjectCopy(t *testing.T, payload []byte) map[string]any {
	t.Helper()
	var document map[string]any
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func TestReviewProviderArtifactSchemasAreStrictAndBound(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration", "v1", "schemas")
	tests := []struct {
		name string
		id   string
	}{
		{name: "artifact-subject.schema.json", id: "https://gentle-ai.dev/contracts/review-integration/v1/schemas/artifact-subject.schema.json"},
		{name: "admitted-result.schema.json", id: "https://gentle-ai.dev/contracts/review-integration/v1/schemas/admitted-result.schema.json"},
		{name: "correction-plan-request.schema.json", id: reviewtransaction.CorrectionPlanRequestSchemaID},
		{name: "result-artifact-v2.schema.json", id: "https://gentle-ai.dev/contracts/review-integration/v1/schemas/result-artifact-v2.schema.json"},
		{name: "start-v2.schema.json", id: ReviewIntegrationStartSchemaIDV2},
		{name: "status-v2.schema.json", id: ReviewIntegrationStatusSchemaIDV2},
		{name: "transition-execution.schema.json", id: "https://gentle-ai.dev/contracts/review-integration/v1/schemas/transition-execution.schema.json"},
		{name: "authority-repair-assessment.schema.json", id: reviewtransaction.AuthorityRepairAssessmentSchemaID},
		{name: "repair.schema.json", id: ReviewIntegrationRepairSchemaID},
	}
	documents := make(map[string]map[string]any, len(tests))
	for _, tt := range tests {
		payload, err := os.ReadFile(filepath.Join(root, tt.name))
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(payload, &schema); err != nil {
			t.Fatal(err)
		}
		if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" || schema["$id"] != tt.id || schema["additionalProperties"] != false {
			t.Fatalf("%s header = %#v", tt.name, schema)
		}
		documents[tt.name] = schema
	}

	artifact := documents["result-artifact-v2.schema.json"]
	artifactRequired := schemaStringArray(t, artifact["required"])
	for _, field := range []string{"subject_hash", "admission_decision"} {
		if !slices.Contains(artifactRequired, field) {
			t.Fatalf("result artifact v2 omits required %q: %v", field, artifactRequired)
		}
	}
	if artifact["oneOf"] == nil {
		t.Fatal("result artifact v2 does not require exactly one provider-owned locator")
	}

	start := documents["start-v2.schema.json"]
	if !slices.Contains(schemaStringArray(t, start["required"]), "artifact_subjects") {
		t.Fatal("START v2 does not require provider-owned artifact subjects")
	}
	riskCodes := start["$defs"].(map[string]any)["risk_reason"].(map[string]any)["properties"].(map[string]any)["code"].(map[string]any)["enum"]
	codes := schemaStringArray(t, riskCodes)
	for _, code := range []string{string(reviewtransaction.RiskReasonProcessBoundary), string(reviewtransaction.RiskReasonProcessScanLimit)} {
		if !slices.Contains(codes, code) {
			t.Fatalf("START v2 rejects runtime risk reason %q: %v", code, codes)
		}
	}
	startStates := schemaStringArray(t, start["properties"].(map[string]any)["state"].(map[string]any)["enum"])
	for _, state := range []string{string(reviewtransaction.StateCorrectionRequired), string(reviewtransaction.StateValidating)} {
		if !slices.Contains(startStates, state) {
			t.Fatalf("START v2 rejects valid compact state %q: %v", state, startStates)
		}
	}

	status := documents["status-v2.schema.json"]
	transitionExecution := documents["transition-execution.schema.json"]
	transitionArtifact := transitionExecution["$defs"].(map[string]any)["transition_artifact"].(map[string]any)
	transitionRequired := schemaStringArray(t, transitionArtifact["required"])
	for _, field := range []string{"subject_hash", "admission_decision"} {
		if !slices.Contains(transitionRequired, field) {
			t.Fatalf("status v2 transition artifact omits %q: %v", field, transitionRequired)
		}
	}
	properties := transitionArtifact["properties"].(map[string]any)
	if properties["schema"].(map[string]any)["const"] != reviewResultArtifactSchema ||
		properties["admission_decision"].(map[string]any)["const"] != string(reviewtransaction.ArtifactAdmissionCompleted) {
		t.Fatalf("status v2 artifact identity = %#v", properties)
	}
	transitionInput := status["$defs"].(map[string]any)["transition_input"].(map[string]any)
	inputRules := transitionInput["allOf"].([]any)
	captureRule := inputRules[1].(map[string]any)
	captureThen := captureRule["then"].(map[string]any)
	for _, field := range []string{"artifact_subject", "candidate_diff", "changed_path_manifest"} {
		if !slices.Contains(schemaStringArray(t, captureThen["required"]), field) {
			t.Fatalf("legacy status v2 capture input omits required frozen context %q: %#v", field, captureThen)
		}
	}
	inputProperties := transitionInput["properties"].(map[string]any)
	if inputProperties["artifact_subject"].(map[string]any)["$ref"] != "artifact-subject.schema.json" ||
		inputProperties["candidate_diff"] == nil || inputProperties["base_tree"] != nil || inputProperties["candidate_tree"] != nil ||
		inputProperties["changed_path_manifest"].(map[string]any)["type"] != "array" {
		t.Fatalf("legacy status v2 capture input frozen context = %#v", inputProperties)
	}

	v2Root := filepath.Join("..", "..", "contracts", "review-integration", "v2", "schemas")
	v2Schemas := []struct {
		name string
		id   string
	}{
		{name: "artifact-subject.schema.json", id: "https://gentle-ai.dev/contracts/review-integration/v2/schemas/artifact-subject.schema.json"},
		{name: "admitted-result.schema.json", id: "https://gentle-ai.dev/contracts/review-integration/v2/schemas/admitted-result.schema.json"},
		{name: "start.schema.json", id: ReviewIntegrationStartSchemaIDV3},
		{name: "start-v4.schema.json", id: ReviewIntegrationStartSchemaIDV4},
		{name: "status.schema.json", id: ReviewIntegrationStatusSchemaIDV3},
		{name: "status-v4.schema.json", id: ReviewIntegrationStatusSchemaIDV4},
		{name: "status-v5.schema.json", id: ReviewIntegrationStatusSchemaIDV5},
		{name: "status-v6.schema.json", id: ReviewIntegrationStatusSchemaIDV6},
		{name: "status-v7.schema.json", id: ReviewIntegrationStatusSchemaIDV7},
		{name: "status-v8.schema.json", id: ReviewIntegrationStatusSchemaIDV8},
		{name: "status-v9.schema.json", id: ReviewIntegrationStatusSchemaIDV9},
		{name: "capabilities.schema.json", id: ReviewIntegrationCapabilitiesSchemaIDV2},
		{name: "capabilities-v2.1.schema.json", id: ReviewIntegrationCapabilitiesSchemaIDV21},
		{name: "capabilities-v2.2.schema.json", id: ReviewIntegrationCapabilitiesSchemaIDV22},
		{name: "capabilities-v2.3.schema.json", id: ReviewIntegrationCapabilitiesSchemaIDV23},
		{name: "capabilities-v2.4.schema.json", id: ReviewIntegrationCapabilitiesSchemaIDV24},
		{name: "capabilities-v2.5.schema.json", id: ReviewIntegrationCapabilitiesSchemaIDV25},
		{name: "capabilities-v2.6.schema.json", id: ReviewIntegrationCapabilitiesSchemaIDV26},
		{name: "intended-untracked-selection.schema.json", id: reviewIntendedUntrackedSelectionSchema},
		{name: "consent.schema.json", id: ReviewIntegrationConsentSchemaIDV2},
		{name: "consent-v3.schema.json", id: ReviewIntegrationConsentSchemaIDV3},
		{name: "failure.schema.json", id: ReviewIntegrationFailureSchemaIDV2},
		{name: "operation.schema.json", id: ReviewIntegrationOperationSchemaIDV2},
		{name: "repair.schema.json", id: ReviewIntegrationRepairSchemaIDV2},
		{name: "gate-result.schema.json", id: "https://gentle-ai.dev/contracts/review-integration/v2/schemas/gate-result.schema.json"},
		{name: "last-event-closure.schema.json", id: "https://gentle-ai.dev/contracts/review-integration/v2/schemas/last-event-closure.schema.json"},
		{name: "opencode-provider-role.schema.json", id: "https://gentle-ai.dev/contracts/review-integration/v2/schemas/opencode-provider-role.schema.json"},
	}
	v2Documents := make(map[string]map[string]any, len(v2Schemas))
	for _, tt := range v2Schemas {
		payload, err := os.ReadFile(filepath.Join(v2Root, tt.name))
		if err != nil {
			t.Fatal(err)
		}
		var schema map[string]any
		if err := json.Unmarshal(payload, &schema); err != nil {
			t.Fatal(err)
		}
		if schema["$schema"] != "https://json-schema.org/draft/2020-12/schema" || schema["$id"] != tt.id || schema["additionalProperties"] != false {
			t.Fatalf("v2 %s header = %#v", tt.name, schema)
		}
		v2Documents[tt.name] = schema
	}
	v2Input := v2Documents["status.schema.json"]["$defs"].(map[string]any)["transition_input"].(map[string]any)
	v2CaptureThen := v2Input["allOf"].([]any)[1].(map[string]any)["then"].(map[string]any)
	for _, field := range []string{"artifact_subject", "base_tree", "candidate_tree", "changed_path_manifest"} {
		if !slices.Contains(schemaStringArray(t, v2CaptureThen["required"]), field) {
			t.Fatalf("native Git status capture input omits %q: %#v", field, v2CaptureThen)
		}
	}
	v2Properties := v2Input["properties"].(map[string]any)
	if v2Properties["candidate_diff"] != nil || v2Properties["base_tree"] == nil || v2Properties["candidate_tree"] == nil {
		t.Fatalf("native Git status capture input = %#v", v2Properties)
	}
}

func TestReviewProviderArtifactV2FixturesValidate(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration", "v2", "fixtures")
	startPayload, err := os.ReadFile(filepath.Join(root, "start.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var start ReviewIntegrationStartResult
	if err := json.Unmarshal(startPayload, &start); err != nil {
		t.Fatal(err)
	}
	if err := start.Validate(); err != nil {
		t.Fatalf("v2 START fixture: %v", err)
	}
	startV4Payload, err := os.ReadFile(filepath.Join(root, "start-v4.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var startV4 ReviewIntegrationStartResult
	if err := json.Unmarshal(startV4Payload, &startV4); err != nil {
		t.Fatal(err)
	}
	if err := startV4.Validate(); err != nil {
		t.Fatalf("v4 START fixture: %v", err)
	}
	if startV4.NextTransition == nil || startV4.NextTransition.Execute == nil ||
		startV4.NextTransition.Execute.Operation != "review.status" {
		t.Fatalf("v4 START fixture continuation = %#v", startV4.NextTransition)
	}
	statusPayload, err := os.ReadFile(filepath.Join(root, "status.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var status ReviewTargetStatusResult
	if err := json.Unmarshal(statusPayload, &status); err != nil {
		t.Fatal(err)
	}
	if err := status.Validate(); err != nil {
		t.Fatalf("v2 STATUS fixture: %v", err)
	}
	v5StatusPayload, err := os.ReadFile(filepath.Join(root, "status-v5.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v5Status ReviewTargetStatusResult
	if err := json.Unmarshal(v5StatusPayload, &v5Status); err != nil {
		t.Fatal(err)
	}
	if err := v5Status.Validate(); err != nil {
		t.Fatalf("v5 STATUS fixture: %v", err)
	}
	consentPayload, err := os.ReadFile(filepath.Join(root, "consent.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var consent ReviewIntegrationConsentResult
	if err := json.Unmarshal(consentPayload, &consent); err != nil {
		t.Fatal(err)
	}
	if err := consent.Validate(); err != nil {
		t.Fatalf("v2 consent fixture: %v", err)
	}
	consentV3Payload, err := os.ReadFile(filepath.Join(root, "consent-v3.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var consentV3 ReviewIntegrationConsentResult
	if err := json.Unmarshal(consentV3Payload, &consentV3); err != nil {
		t.Fatal(err)
	}
	if err := consentV3.Validate(); err != nil || consentV3.Agent != "claude-code" {
		t.Fatalf("v2.1 consent fixture: %#v, %v", consentV3, err)
	}
}

func schemaStringArray(t *testing.T, value any) []string {
	t.Helper()
	values, ok := value.([]any)
	if !ok {
		t.Fatalf("schema value is not an array: %#v", value)
	}
	result := make([]string, len(values))
	for index, value := range values {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("schema array value is not a string: %#v", value)
		}
		result[index] = text
	}
	return result
}
