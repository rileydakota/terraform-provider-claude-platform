package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// TestProviderSchemas round-trips every resource and data source schema
// through the protocol server, which runs the framework's schema validation
// (invalid attribute combinations fail here without any API credentials).
func TestProviderSchemas(t *testing.T) {
	srv, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatalf("creating provider server: %v", err)
	}

	resp, err := srv.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("schema diagnostic: %s: %s", d.Summary, d.Detail)
		}
	}

	wantResources := []string{
		"claudeplatform_workspace",
		"claudeplatform_workspace_member",
		"claudeplatform_organization_invite",
		"claudeplatform_service_account",
		"claudeplatform_service_account_workspace",
		"claudeplatform_federation_issuer",
		"claudeplatform_federation_rule",
	}
	for _, name := range wantResources {
		if _, ok := resp.ResourceSchemas[name]; !ok {
			t.Errorf("missing resource schema %q", name)
		}
	}

	wantDataSources := []string{
		"claudeplatform_organization",
		"claudeplatform_workspace",
		"claudeplatform_workspaces",
		"claudeplatform_user",
	}
	for _, name := range wantDataSources {
		if _, ok := resp.DataSourceSchemas[name]; !ok {
			t.Errorf("missing data source schema %q", name)
		}
	}
}
