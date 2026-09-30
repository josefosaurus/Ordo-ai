package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/reviewtransaction"
)

func TestReviewCapabilitiesV13AdvertisesProviderAdmissionAndRecovery(t *testing.T) {
	surface := reviewCapabilitiesStaticSurface()
	for _, want := range []ReviewCapabilityFeature{
		{Name: "opaque_repository_context", Supported: true, Requires: []string{"compact_v2_authority", "native_next_transition"}},
		{Name: "provider_artifact_admission", Supported: true, Requires: []string{"compact_v2_authority", "native_frozen_candidate_context", "opaque_repository_context"}},
		{Name: "provider_targeted_validation_request", Supported: true, Requires: []string{"compact_v2_authority", "native_next_transition"}},
		{Name: "recovered_correction_evidence", Supported: true, Requires: []string{"compact_v2_authority", "provider_targeted_validation_request"}},
		{Name: "validating_result_reopen", Supported: true, Requires: []string{"compact_v2_authority", "provider_artifact_admission"}},
	} {
		if !slices.ContainsFunc(surface.Features.Optional, func(got ReviewCapabilityFeature) bool {
			return got.Name == want.Name && got.Supported == want.Supported && slices.Equal(got.Requires, want.Requires)
		}) {
			t.Fatalf("v1.3 optional capabilities missing %#v: %#v", want, surface.Features.Optional)
		}
	}
	for _, schema := range []string{
		reviewtransaction.ArtifactSubjectSchemaV1,
		reviewtransaction.AdmittedReviewerResultSchemaV1,
		reviewtransaction.TargetedValidationRequestSchema,
	} {
		if !slices.Contains(surface.Schemas, schema) {
			t.Fatalf("v1.3 schemas do not advertise %q: %v", schema, surface.Schemas)
		}
	}
}

func TestReviewCapabilitiesV10ThroughV12ArtifactsRemainByteIdentical(t *testing.T) {
	root := filepath.Join("..", "..", "contracts", "review-integration", "v1")
	want := map[string]string{
		"fixtures/capabilities.fixture.json": "b3ca822189a236f2d891628c665ca23e308bf5185a1701e1f07231bd970461bb",
		// Ordo: command renamed to ordo.
		"fixtures/capabilities-v1.1.fixture.json": "980240c4c760683717efb9fe01eb495fc97afce72cf246068ae43a3489f4000c",
		// Ordo: command renamed to ordo.
		"fixtures/capabilities-v1.2.fixture.json": "1177ff6db81b3edef6ce17049e19375a9245ed8b6b8e4b01d8c70f0fcf8ab298",
		"schemas/capabilities.schema.json":        "ad333177494a251beac153f74bd751fa77126a9968aad69e64fc2abf15cff0f7",
		// Ordo: command renamed to ordo.
		"schemas/capabilities-v1.1.schema.json": "befc1d0b89a7d5ddd260c0bc32519ec7005fd14f4c8fc066e5f12248a7e3784b",
		// Ordo: command renamed to ordo.
		"schemas/capabilities-v1.2.schema.json": "aa05843c639c9ec6ddaf9924d2bbe8e12e7f14930661049ad0971cac1036a50b",
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
