package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ronhuafeng/codexsdk-go/codexsdk/internal/protocolgen"
)

const defaultSchemaRoot = "codexsdk/internal/protocolschema/appserver/v2"

func main() {
	schemaRootFlag := flag.String("schema-root", "", "checked-in app-server v2 schema baseline root; defaults to the manifest directory or the checked-in baseline")
	manifestPathFlag := flag.String("manifest", "", "classified app-server v2 manifest path; defaults to <schema-root>/manifest.json")
	outDir := flag.String("out", "codexsdk/protocolv2", "protocolv2 output directory")
	flag.Parse()

	schemaRoot, manifestPath := resolveSchemaInputs(*schemaRootFlag, *manifestPathFlag)

	typePlan, err := protocolgen.BuildProtocolTypePlan(schemaRoot)
	if err != nil {
		fatal(err)
	}
	manifest, err := protocolgen.LoadManifest(manifestPath)
	if err != nil {
		fatal(err)
	}
	generated, err := protocolgen.GenerateMethodRegistry(manifest)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*outDir, "method_registry.gen.go"), generated, 0o644); err != nil {
		fatal(err)
	}
	generatedTypes, err := protocolgen.GenerateProtocolTypes(typePlan)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*outDir, "protocol_types.gen.go"), generatedTypes, 0o644); err != nil {
		fatal(err)
	}
}

func resolveSchemaInputs(schemaRootFlag, manifestPathFlag string) (schemaRoot string, manifestPath string) {
	switch {
	case schemaRootFlag != "":
		schemaRoot = schemaRootFlag
	case manifestPathFlag != "":
		schemaRoot = filepath.Dir(manifestPathFlag)
	default:
		schemaRoot = defaultSchemaRoot
	}
	if manifestPathFlag != "" {
		manifestPath = manifestPathFlag
	} else {
		manifestPath = filepath.Join(schemaRoot, "manifest.json")
	}
	return schemaRoot, manifestPath
}

func fatal(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "protocolv2gen: %v\n", err)
	os.Exit(1)
}
