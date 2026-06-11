package main

import "testing"

func TestResolveSchemaInputsUsesManifestDirectoryWhenSchemaRootOmitted(t *testing.T) {
	schemaRoot, manifestPath := resolveSchemaInputs("", "/tmp/appserver/v2/manifest.json")
	if schemaRoot != "/tmp/appserver/v2" {
		t.Fatalf("schemaRoot = %q, want /tmp/appserver/v2", schemaRoot)
	}
	if manifestPath != "/tmp/appserver/v2/manifest.json" {
		t.Fatalf("manifestPath = %q, want /tmp/appserver/v2/manifest.json", manifestPath)
	}
}

func TestResolveSchemaInputsHonorsExplicitSchemaRoot(t *testing.T) {
	schemaRoot, manifestPath := resolveSchemaInputs("/tmp/schema-root", "/tmp/custom/manifest.json")
	if schemaRoot != "/tmp/schema-root" {
		t.Fatalf("schemaRoot = %q, want /tmp/schema-root", schemaRoot)
	}
	if manifestPath != "/tmp/custom/manifest.json" {
		t.Fatalf("manifestPath = %q, want /tmp/custom/manifest.json", manifestPath)
	}
}
