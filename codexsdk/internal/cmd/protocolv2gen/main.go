package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ronhuafeng/codexsdk-go/codexsdk/internal/protocolgen"
)

const defaultSchemaRoot = "codexsdk/internal/protocolschema/appserver/v2"

func main() {
	schemaRootFlag := flag.String("schema-root", "", "checked-in app-server v2 schema baseline root; defaults to the manifest directory or the checked-in baseline")
	manifestPathFlag := flag.String("manifest", "", "classified app-server v2 manifest path; defaults to <schema-root>/manifest.json")
	outDir := flag.String("out", "codexsdk/protocolv2", "protocolv2 output directory")
	stdout := flag.String("stdout", "", "write one generated artifact to stdout instead of files: method-registry or protocol-types")
	flag.Parse()

	if err := run(*schemaRootFlag, *manifestPathFlag, *outDir, *stdout, os.Stdout); err != nil {
		fatal(err)
	}
}

func run(schemaRootFlag, manifestPathFlag, outDir, stdout string, writer io.Writer) error {
	schemaRoot, manifestPath := resolveSchemaInputs(schemaRootFlag, manifestPathFlag)
	switch stdout {
	case "":
	case "method-registry":
		methodRegistry, err := generateMethodRegistry(manifestPath)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, bytes.NewReader(methodRegistry))
		return err
	case "protocol-types":
		protocolTypes, err := generateProtocolTypes(schemaRoot)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, bytes.NewReader(protocolTypes))
		return err
	default:
		return fmt.Errorf("-stdout must be method-registry or protocol-types")
	}

	methodRegistry, err := generateMethodRegistry(manifestPath)
	if err != nil {
		return err
	}
	protocolTypes, err := generateProtocolTypes(schemaRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "method_registry.gen.go"), methodRegistry, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "protocol_types.gen.go"), protocolTypes, 0o644); err != nil {
		return err
	}
	return nil
}

func generateMethodRegistry(manifestPath string) ([]byte, error) {
	manifest, err := protocolgen.LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	return protocolgen.GenerateMethodRegistry(manifest)
}

func generateProtocolTypes(schemaRoot string) ([]byte, error) {
	typePlan, err := protocolgen.BuildProtocolTypePlan(schemaRoot)
	if err != nil {
		return nil, err
	}
	return protocolgen.GenerateProtocolTypes(typePlan)
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
