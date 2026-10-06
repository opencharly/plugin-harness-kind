package harnesskind

// marketplace_input_test.go — coverage for the `marketplace` kind's `version`
// OPTIONALITY in the served self-contained #MarketplaceInput schema.
//
// WHY THIS TEST EXISTS. The org-wide cutover ("drop the retired config version:
// stamp") removed the `version:` stamp from every authored `marketplace:` entity,
// because the marketplace versions by commit SHA (marketplace/README.md: "No
// `version` fields anywhere"). The cutover did NOT update this schema, so
// `#MarketplaceInput.version` stayed REQUIRED and the one repo carrying a
// `marketplace:` entity failed its own `charly box validate` gate with
// `#MarketplaceInput.version: incomplete value`.
//
// Case (a) FAILS on the pre-change schema — it is the reason this change exists.
// Case (b) pins back-compat, and (c) pins that optionality did not become
// "anything goes".
//
// It compiles the SAME served schema the load gate compiles, through the SDK's
// own validator (sdk.NewSchemaValidator over the embedded schemaFS), so the test
// exercises the real gate rather than a hand-rolled copy.

import (
	"testing"

	"github.com/opencharly/sdk"
)

func TestMarketplaceInputVersionIsOptional(t *testing.T) {
	v, err := sdk.NewSchemaValidator(schemaFS, "schema")
	if err != nil {
		t.Fatalf("compile the served schema: %v", err)
	}

	// (a) THE NEW BEHAVIOUR — a version-less marketplace body must validate.
	noVersion := []byte(`{
		"name": "charly-plugins",
		"families": {"core": {"category": "commands"}}
	}`)
	if err := v.ValidateJSON("#MarketplaceInput", noVersion); err != nil {
		t.Fatalf("#MarketplaceInput rejected a version-less marketplace body (the cutover's own shape):\n%v", err)
	}
	t.Log("version-less marketplace body: ACCEPTED (the stamp is optional)")

	// (b) BACK-COMPAT — a legacy body carrying a CalVer version must still validate.
	withVersion := []byte(`{
		"name": "charly-plugins",
		"version": "2026.272.0616",
		"families": {"core": {"category": "commands"}}
	}`)
	if err := v.ValidateJSON("#MarketplaceInput", withVersion); err != nil {
		t.Fatalf("#MarketplaceInput rejected a legacy version-bearing body:\n%v", err)
	}
	t.Log("legacy version-bearing marketplace body: ACCEPTED (back-compat held)")

	// (c) The CalVer pattern survives — a malformed version is still rejected, so
	// optionality did not weaken the field's own constraint.
	badVersion := []byte(`{
		"name": "charly-plugins",
		"version": "not-a-calver",
		"families": {"core": {"category": "commands"}}
	}`)
	if err := v.ValidateJSON("#MarketplaceInput", badVersion); err == nil {
		t.Fatal("#MarketplaceInput ACCEPTED a malformed version — the CalVer pattern was lost")
	}
	t.Log("malformed version: REJECTED (the CalVer pattern survives)")
}
