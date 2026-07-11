package codexsdk

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ronhuafeng/codexsdk-go/codexsdk/protocolv2"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewValidationInitializeAndExactCommand(t *testing.T) {
	cwd := t.TempDir()
	if _, err := New(ClientOptions{CWD: "", Command: fakeCommand("happy")}); err == nil {
		t.Fatal("New without cwd succeeded")
	}
	notDir := filepath.Join(cwd, "not-dir")
	if err := os.WriteFile(notDir, []byte("not a dir"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(ClientOptions{CWD: notDir, Command: fakeCommand("happy")}); err == nil {
		t.Fatal("New with non-directory cwd succeeded")
	}
	if _, err := New(ClientOptions{CWD: cwd}); err == nil {
		t.Fatal("New without command succeeded")
	}
	if _, err := New(ClientOptions{CWD: cwd, Command: []string{" "}}); err == nil {
		t.Fatal("New with empty argv[0] succeeded")
	}
	if _, err := New(ClientOptions{CWD: cwd, Command: []string{filepath.Join(cwd, "missing-codex-app-server")}}); err == nil {
		t.Fatal("New with missing executable succeeded")
	}

	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	client, err := New(ClientOptions{
		CWD:         cwd,
		Command:     fakeCommand("happy", "literal arg"),
		ClientName:  "unit-client",
		ClientTitle: "Unit Client",
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	defer client.Close()

	records := readRecords(t, record)
	if got := records[0]["cwd"]; got != cwd {
		t.Fatalf("cwd = %q, want %q", got, cwd)
	}
	argv, _ := records[0]["argv"].([]any)
	gotArgv := stringify(argv)
	wantSuffix := []string{"-test.run=TestHelperProcess", "--", "happy", "literal arg"}
	if !reflect.DeepEqual(gotArgv[len(gotArgv)-len(wantSuffix):], wantSuffix) {
		t.Fatalf("argv suffix = %#v, want %#v", gotArgv, wantSuffix)
	}
	if contains(gotArgv, "app-server") || contains(gotArgv, "--listen") || contains(gotArgv, "stdio://") {
		t.Fatalf("SDK appended app-server arguments: %#v", gotArgv)
	}
	init := firstRecord(records, "recv", "initialize")
	clientInfo := init["params"].(map[string]any)["clientInfo"].(map[string]any)
	if clientInfo["name"] != "unit-client" || clientInfo["title"] != "Unit Client" {
		t.Fatalf("initialize clientInfo = %#v", clientInfo)
	}
	capabilities := init["params"].(map[string]any)["capabilities"].(map[string]any)
	if _, ok := capabilities["experimentalApi"]; ok {
		t.Fatalf("experimentalApi enabled by default: %#v", capabilities)
	}
	initialized := firstRecord(records, "recv", "initialized")
	if initialized["params"] != nil {
		t.Fatalf("initialized notification params = %#v, want omitted/null", initialized["params"])
	}
}

func TestTextAndFilesPreservesBlankPathForValidation(t *testing.T) {
	items := TextAndFiles("prompt", []string{""})
	if len(items) != 2 || items[1].Type != InputItemFile || items[1].Path != "" {
		t.Fatalf("TextAndFiles silently discarded blank path: %#v", items)
	}
}

func TestNewExperimentalCapabilityOptIn(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	client, err := New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("happy"),
		Capabilities: ClientCapabilities{
			ExperimentalAPI: true,
		},
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	defer client.Close()

	init := firstRecord(readRecords(t, record), "recv", "initialize")
	capabilities := init["params"].(map[string]any)["capabilities"].(map[string]any)
	if capabilities["experimentalApi"] != true {
		t.Fatalf("initialize capabilities = %#v, want experimentalApi true", capabilities)
	}
}

func TestNewUsesExactGeneratedInitializeParams(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	experimental := true
	capabilities := protocolv2.InitializeCapabilities{ExperimentalAPI: &experimental}
	options := ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("happy"),
		Initialize: protocolv2.InitializeParams{
			Capabilities: protocolv2.Value(capabilities),
			ClientInfo: protocolv2.ClientInfo{
				Name:    "exact-client",
				Title:   protocolv2.Value("Exact Client"),
				Version: "v0.2-test",
			},
		},
	}
	client, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	experimental = false
	defer client.Close()
	init := firstRecord(readRecords(t, record), "recv", protocolv2.MethodInitialize)
	params := init["params"].(map[string]any)
	info := params["clientInfo"].(map[string]any)
	if info["name"] != "exact-client" || info["title"] != "Exact Client" || info["version"] != "v0.2-test" {
		t.Fatalf("initialize params = %#v", params)
	}
	if params["capabilities"].(map[string]any)["experimentalApi"] != true {
		t.Fatalf("initialize capabilities = %#v", params["capabilities"])
	}
}

func TestNewRejectsMalformedInitializeResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	_, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("bad-initialize")})
	if err == nil {
		t.Fatal("New succeeded with malformed initialize response")
	}
	if !strings.Contains(err.Error(), "initialize response invalid") ||
		!strings.Contains(err.Error(), "decode InitializeResponse.platformOs: missing required field") {
		t.Fatalf("malformed initialize error = %v", err)
	}
}

func TestNewPropagatesInitializeProtocolError(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	_, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("initialize-error")})
	if err == nil {
		t.Fatal("New succeeded with initialize protocol error")
	}
	if !strings.Contains(err.Error(), "app-server error") ||
		!strings.Contains(err.Error(), "-32000") ||
		!strings.Contains(err.Error(), "initialize failed") {
		t.Fatalf("initialize protocol error = %v", err)
	}
}

func TestNewFailsWhenAppServerClosesStdoutDuringInitialize(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	_, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("initialize-close-stdout")})
	if err == nil {
		t.Fatal("New succeeded after app-server closed stdout during initialize")
	}
	if !strings.Contains(err.Error(), "app-server closed stdout") {
		t.Fatalf("initialize stdout close error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", "initialize") == nil {
		t.Fatal("fake app-server did not receive initialize before closing stdout")
	}
}

func TestNewFailsOnMalformedStdoutDuringInitializeWithoutLeakingRawLine(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	_, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("initialize-malformed-stdout")})
	if err == nil {
		t.Fatal("New succeeded after malformed stdout during initialize")
	}
	if !strings.Contains(err.Error(), "invalid app-server JSON-RPC line bytes=") ||
		!strings.Contains(err.Error(), "sha256=") {
		t.Fatalf("initialize malformed stdout error missing sanitized metadata: %v", err)
	}
	for _, forbidden := range []string{"initialize_secret", "transcript", "{not-json"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("initialize malformed stdout error leaked raw content %q: %v", forbidden, err)
		}
	}
}

func TestRootClientCloseWaitsForAppServerProcess(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	c := root.(*client)
	if c.cmd == nil || c.cmd.Process == nil {
		t.Fatal("root client did not retain SDK-owned app-server process")
	}
	if c.cmd.ProcessState != nil {
		t.Fatalf("process state before Close = %v, want nil", c.cmd.ProcessState)
	}

	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	if c.cmd.ProcessState == nil {
		t.Fatal("Close returned before waiting for app-server process")
	}
	if err := root.Close(); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
	if _, err := c.call(context.Background(), "after/close", nil); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("post-close call error = %v, want ErrClientClosed", err)
	}
}

func TestRootClientInterfaceDoesNotExposeThreadLifecycleOrRawCall(t *testing.T) {
	root := reflect.TypeOf((*Client)(nil)).Elem()
	for _, name := range []string{
		"Call",
		"ForkThread",
		"ResumeThread",
		"ResumeThreadStream",
		"StartThread",
		"StartThreadStream",
	} {
		if _, ok := root.MethodByName(name); ok {
			t.Fatalf("root Client exposes %s; use typed facades or ThreadClient accessor instead", name)
		}
	}
	if _, ok := root.MethodByName("ThreadClient"); !ok {
		t.Fatal("root Client missing ThreadClient accessor")
	}
	if _, ok := root.MethodByName("Close"); !ok {
		t.Fatal("root Client missing Close")
	}

	threadClient := reflect.TypeOf((*ThreadClient)(nil)).Elem()
	wantThreadClientMethods := map[string]struct{}{
		"ForkThread":         {},
		"ResumeThread":       {},
		"ResumeThreadStream": {},
		"StartThread":        {},
		"StartThreadStream":  {},
	}
	if threadClient.NumMethod() != len(wantThreadClientMethods) {
		t.Fatalf("ThreadClient method count = %d, want %d", threadClient.NumMethod(), len(wantThreadClientMethods))
	}
	for name := range wantThreadClientMethods {
		if _, ok := threadClient.MethodByName(name); !ok {
			t.Fatalf("ThreadClient missing %s method", name)
		}
	}
	if _, ok := threadClient.MethodByName("Close"); ok {
		t.Fatal("ThreadClient exposes Close; root Client owns app-server lifecycle")
	}

	expected := expectedSDKSurfaceMethods(t)
	surface := reflect.TypeOf((*SDKSurface)(nil)).Elem()
	for accessor, methods := range expected {
		if _, ok := root.MethodByName(accessor); !ok {
			t.Fatalf("root Client missing %s facade accessor", accessor)
		}
		surfaceMethod, ok := surface.MethodByName(accessor)
		if !ok {
			t.Fatalf("SDKSurface missing %s accessor", accessor)
		}
		facadeType := surfaceMethod.Type.Out(0)
		if facadeType.NumMethod() != len(methods) {
			t.Fatalf("%s method count = %d, want %d", accessor, facadeType.NumMethod(), len(methods))
		}
		for name := range methods {
			if _, ok := facadeType.MethodByName(name); !ok {
				t.Fatalf("%s missing %s method", accessor, name)
			}
		}
		if _, ok := facadeType.MethodByName("Call"); ok {
			t.Fatalf("%s exposes raw Call", accessor)
		}
	}
}

type sdkSurfaceManifest struct {
	Entries []struct {
		Direction             string `json:"direction"`
		FacadeTarget          string `json:"facade_target"`
		Kind                  string `json:"kind"`
		Method                string `json:"method"`
		ParamsOrPayloadSchema string `json:"params_or_payload_schema"`
		ResponseType          string `json:"response_type"`
	} `json:"entries"`
}

func expectedSDKSurfaceMethods(t *testing.T) map[string]map[string]struct{} {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("internal", "protocolschema", "appserver", "v2", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest sdkSurfaceManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	methods := generatedMethodNames(t)
	types := generatedProtocolTypeNames(t)
	expected := map[string]map[string]struct{}{}
	for _, entry := range manifest.Entries {
		if entry.Direction != "client_to_server" || entry.Kind != "request" {
			continue
		}
		if !methods[entry.Method] {
			continue
		}
		if entry.ParamsOrPayloadSchema != "" && !types[entry.ParamsOrPayloadSchema] {
			continue
		}
		if entry.ResponseType == "" || !types[entry.ResponseType] {
			continue
		}
		prefix, operation, ok := strings.Cut(entry.FacadeTarget, "().")
		if !ok || prefix == "" || operation == "" || strings.Contains(prefix, ".") || strings.Contains(operation, ".") {
			continue
		}
		methods := expected[prefix]
		if methods == nil {
			methods = map[string]struct{}{}
			expected[prefix] = methods
		}
		methods[operation] = struct{}{}
	}
	if len(expected) == 0 {
		t.Fatal("manifest did not declare any SDK facade methods")
	}
	return expected
}

func generatedMethodNames(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("protocolv2", "method_registry.gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	matches := regexp.MustCompile(`(?m)^\s*Method[A-Za-z0-9]+\s+=\s+"([^"]+)"`).FindAllSubmatch(raw, -1)
	methods := map[string]bool{}
	for _, match := range matches {
		methods[string(match[1])] = true
	}
	return methods
}

func generatedProtocolTypeNames(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("protocolv2", "protocol_types.gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	matches := regexp.MustCompile(`(?m)^type\s+([A-Za-z][A-Za-z0-9]*)\b`).FindAllSubmatch(raw, -1)
	types := map[string]bool{}
	for _, match := range matches {
		types[string(match[1])] = true
	}
	return types
}

func TestConfigProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	config := root.Config()

	batch, err := config.BatchWrite(context.Background(), protocolv2.ConfigBatchWriteParams{
		Edits: []protocolv2.ConfigEdit{{
			KeyPath:       "model",
			MergeStrategy: protocolv2.MergeStrategyReplace,
			Value:         protocolv2.JSONString("gpt-5"),
		}},
		ExpectedVersion:  protocolv2.Value("v1"),
		FilePath:         protocolv2.Null[string](),
		ReloadUserConfig: Bool(true),
	})
	if err != nil {
		t.Fatalf("Config().BatchWrite returned error: %v", err)
	}
	if batch.Status != protocolv2.WriteStatusOk || batch.Version != "v2" {
		t.Fatalf("config/batchWrite response = %#v", batch)
	}
	read, err := config.Read(context.Background(), protocolv2.ConfigReadParams{
		CWD:           protocolv2.Value("/workspace"),
		IncludeLayers: Bool(true),
	})
	if err != nil {
		t.Fatalf("Config().Read returned error: %v", err)
	}
	if read.Config.Model == nil || read.Config.Model.Value == nil || *read.Config.Model.Value != "gpt-5" || read.Origins["model"].Version != "v1" {
		t.Fatalf("config/read response = %#v", read)
	}
	if _, err := config.MCPServerReload(context.Background()); err != nil {
		t.Fatalf("Config().MCPServerReload returned error: %v", err)
	}
	value, err := config.ValueWrite(context.Background(), protocolv2.ConfigValueWriteParams{
		ExpectedVersion: protocolv2.Null[string](),
		FilePath:        protocolv2.Value("/home/user/.codex/config.toml"),
		KeyPath:         "features.example",
		MergeStrategy:   protocolv2.MergeStrategyUpsert,
		Value: protocolv2.JSONObject(map[string]protocolv2.JSONValue{
			"enabled": protocolv2.JSONBool(true),
		}),
	})
	if err != nil {
		t.Fatalf("Config().ValueWrite returned error: %v", err)
	}
	if value.FilePath != "/home/user/.codex/config.toml" || value.Status != protocolv2.WriteStatusOk {
		t.Fatalf("config/value/write response = %#v", value)
	}

	records := readRecords(t, record)
	wantMethods := []string{
		protocolv2.MethodConfigBatchWrite,
		protocolv2.MethodConfigRead,
		protocolv2.MethodConfigMCPServerReload,
		protocolv2.MethodConfigValueWrite,
	}
	for _, method := range wantMethods {
		if firstRecord(records, "recv", method) == nil {
			t.Fatalf("missing config facade method %s in records %#v", method, records)
		}
	}
	batchParams := firstRecord(records, "recv", protocolv2.MethodConfigBatchWrite)["params"].(map[string]any)
	batchEdits := batchParams["edits"].([]any)
	batchEdit := batchEdits[0].(map[string]any)
	if batchEdit["keyPath"] != "model" || batchEdit["mergeStrategy"] != "replace" || batchEdit["value"] != "gpt-5" ||
		batchParams["expectedVersion"] != "v1" || batchParams["filePath"] != nil || batchParams["reloadUserConfig"] != true {
		t.Fatalf("config/batchWrite params = %#v", batchParams)
	}
	readParams := firstRecord(records, "recv", protocolv2.MethodConfigRead)["params"].(map[string]any)
	if readParams["cwd"] != "/workspace" || readParams["includeLayers"] != true {
		t.Fatalf("config/read params = %#v", readParams)
	}
	if params := firstRecord(records, "recv", protocolv2.MethodConfigMCPServerReload)["params"]; params != nil {
		t.Fatalf("config/mcpServer/reload params = %#v, want omitted", params)
	}
	valueParams := firstRecord(records, "recv", protocolv2.MethodConfigValueWrite)["params"].(map[string]any)
	configValue := valueParams["value"].(map[string]any)
	if valueParams["expectedVersion"] != nil || valueParams["filePath"] != "/home/user/.codex/config.toml" ||
		valueParams["keyPath"] != "features.example" || valueParams["mergeStrategy"] != "upsert" || configValue["enabled"] != true {
		t.Fatalf("config/value/write params = %#v", valueParams)
	}
}

func TestConfigProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("config-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Config().Read(context.Background(), protocolv2.ConfigReadParams{})
	if err == nil {
		t.Fatal("Config().Read accepted malformed config/read response")
	}
	if !strings.Contains(err.Error(), "decode config/read response") ||
		!strings.Contains(err.Error(), "ConfigReadResponse.origins") {
		t.Fatalf("malformed config/read response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodConfigRead) == nil {
		t.Fatal("config/read was not sent before malformed response decode")
	}
}

func TestConfigRequirementsProtocolFamilyFacadeSendsTypedMethodAndDecodesResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	response, err := root.ConfigRequirements().Read(context.Background())
	if err != nil {
		t.Fatalf("ConfigRequirements().Read returned error: %v", err)
	}
	if response.Requirements == nil || response.Requirements.Value == nil {
		t.Fatalf("configRequirements/read response = %#v", response)
	}
	requirements := response.Requirements.Value
	if requirements.FeatureRequirements == nil || requirements.FeatureRequirements.Value == nil ||
		(*requirements.FeatureRequirements.Value)["alpha"] != true {
		t.Fatalf("configRequirements/read feature requirements = %#v", requirements.FeatureRequirements)
	}
	if requirements.Network == nil || requirements.Network.Value == nil ||
		requirements.Network.Value.Domains == nil || requirements.Network.Value.Domains.Value == nil ||
		(*requirements.Network.Value.Domains.Value)["example.com"] != protocolv2.NetworkDomainPermissionAllow {
		t.Fatalf("configRequirements/read network requirements = %#v", requirements.Network)
	}

	records := readRecords(t, record)
	readRecord := firstRecord(records, "recv", protocolv2.MethodConfigRequirementsRead)
	if readRecord == nil {
		t.Fatalf("missing configRequirements/read in records %#v", records)
	}
	if params := readRecord["params"]; params != nil {
		t.Fatalf("configRequirements/read params = %#v, want omitted", params)
	}
}

func TestConfigRequirementsProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("config-requirements-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.ConfigRequirements().Read(context.Background())
	if err == nil {
		t.Fatal("ConfigRequirements().Read accepted malformed configRequirements/read response")
	}
	if !strings.Contains(err.Error(), "decode configRequirements/read response") ||
		!strings.Contains(err.Error(), `ConfigRequirementsReadResponse: unknown field "extra"`) {
		t.Fatalf("malformed configRequirements/read response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodConfigRequirementsRead) == nil {
		t.Fatal("configRequirements/read was not sent before malformed response decode")
	}
}

func TestExperimentalFeaturesProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	experimentalFeatures := root.ExperimentalFeatures()

	enablement, err := experimentalFeatures.EnablementSet(context.Background(), protocolv2.ExperimentalFeatureEnablementSetParams{
		Enablement: map[string]bool{
			"feature_a": true,
			"feature_b": false,
		},
	})
	if err != nil {
		t.Fatalf("ExperimentalFeatures().EnablementSet returned error: %v", err)
	}
	if enablement.Enablement["feature_a"] != true || enablement.Enablement["feature_b"] != false {
		t.Fatalf("experimentalFeature/enablement/set response = %#v", enablement)
	}
	list, err := experimentalFeatures.List(context.Background(), protocolv2.ExperimentalFeatureListParams{
		Cursor: protocolv2.Null[string](),
		Limit:  protocolv2.Value(uint32(25)),
	})
	if err != nil {
		t.Fatalf("ExperimentalFeatures().List returned error: %v", err)
	}
	if len(list.Data) != 1 || list.Data[0].Name != "feature_a" || list.Data[0].Stage != protocolv2.ExperimentalFeatureStageBeta ||
		list.NextCursor == nil || list.NextCursor.Value == nil || *list.NextCursor.Value != "cursor-2" {
		t.Fatalf("experimentalFeature/list response = %#v", list)
	}

	records := readRecords(t, record)
	setParams := firstRecord(records, "recv", protocolv2.MethodExperimentalFeatureEnablementSet)["params"].(map[string]any)
	setEnablement := setParams["enablement"].(map[string]any)
	if setEnablement["feature_a"] != true || setEnablement["feature_b"] != false {
		t.Fatalf("experimentalFeature/enablement/set params = %#v", setParams)
	}
	listParams := firstRecord(records, "recv", protocolv2.MethodExperimentalFeatureList)["params"].(map[string]any)
	if listParams["cursor"] != nil || listParams["limit"] != float64(25) {
		t.Fatalf("experimentalFeature/list params = %#v", listParams)
	}
}

func TestExperimentalFeaturesProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("experimental-features-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.ExperimentalFeatures().List(context.Background(), protocolv2.ExperimentalFeatureListParams{})
	if err == nil {
		t.Fatal("ExperimentalFeatures().List accepted malformed experimentalFeature/list response")
	}
	if !strings.Contains(err.Error(), "decode experimentalFeature/list response") ||
		!strings.Contains(err.Error(), "ExperimentalFeatureListResponse.data") {
		t.Fatalf("malformed experimentalFeature/list response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodExperimentalFeatureList) == nil {
		t.Fatal("experimentalFeature/list was not sent before malformed response decode")
	}
}

func TestCollaborationModesProtocolFamilyFacadeSendsTypedMethodAndDecodesResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	response, err := root.CollaborationModes().List(context.Background(), protocolv2.CollaborationModeListParams{})
	if err != nil {
		t.Fatalf("CollaborationModes().List returned error after experimental opt-in: %v", err)
	}
	if len(response.Data) != 2 || response.Data[0].Name != "Plan" || response.Data[1].Name != "Default" {
		t.Fatalf("collaborationMode/list response = %#v", response)
	}
	if response.Data[0].Mode == nil || response.Data[0].Mode.Value == nil || *response.Data[0].Mode.Value != protocolv2.ModeKindPlan {
		t.Fatalf("collaboration mode response mode = %#v", response.Data[0].Mode)
	}
	if response.Data[0].Model == nil || response.Data[0].Model.Value != nil {
		t.Fatalf("collaboration mode response model = %#v, want explicit null", response.Data[0].Model)
	}
	if response.Data[0].ReasoningEffort == nil || response.Data[0].ReasoningEffort.Value == nil || *response.Data[0].ReasoningEffort.Value != protocolv2.ReasoningEffort("medium") {
		t.Fatalf("collaboration mode response reasoning effort = %#v", response.Data[0].ReasoningEffort)
	}

	params := firstRecord(readRecords(t, record), "recv", protocolv2.MethodCollaborationModeList)["params"].(map[string]any)
	if len(params) != 0 {
		t.Fatalf("collaborationMode/list params = %#v, want empty object", params)
	}
}

func TestCollaborationModesProtocolFamilyFacadeRejectsExperimentalMethodBeforeWriteUnlessOptedIn(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.CollaborationModes().List(context.Background(), protocolv2.CollaborationModeListParams{})
	if err == nil {
		t.Fatal("CollaborationModes().List accepted experimental method without opt-in")
	}
	if !strings.Contains(err.Error(), "experimental app-server method \"collaborationMode/list\" requires ClientCapabilities.ExperimentalAPI") {
		t.Fatalf("experimental collaborationMode/list error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodCollaborationModeList) != nil {
		t.Fatal("collaborationMode/list was sent after experimental method guard failure")
	}
}

func TestCollaborationModesProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("collaboration-modes-malformed-response"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.CollaborationModes().List(context.Background(), protocolv2.CollaborationModeListParams{})
	if err == nil {
		t.Fatal("CollaborationModes().List accepted malformed collaborationMode/list response")
	}
	if !strings.Contains(err.Error(), "decode collaborationMode/list response") ||
		!strings.Contains(err.Error(), "CollaborationModeListResponse.data") {
		t.Fatalf("malformed collaborationMode/list response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodCollaborationModeList) == nil {
		t.Fatal("collaborationMode/list was not sent before malformed response decode")
	}
}

func TestMemoryProtocolFamilyFacadeSendsTypedMethodAndDecodesResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	if _, err := root.Memory().Reset(context.Background()); err != nil {
		t.Fatalf("Memory().Reset returned error after experimental opt-in: %v", err)
	}
	resetRecord := firstRecord(readRecords(t, record), "recv", protocolv2.MethodMemoryReset)
	if resetRecord == nil {
		t.Fatal("memory/reset was not sent after experimental opt-in")
	}
	if params := resetRecord["params"]; params != nil {
		t.Fatalf("memory/reset params = %#v, want omitted", params)
	}
}

func TestMemoryProtocolFamilyFacadeRejectsExperimentalMethodBeforeWriteUnlessOptedIn(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Memory().Reset(context.Background())
	if err == nil {
		t.Fatal("Memory().Reset accepted experimental method without opt-in")
	}
	if !strings.Contains(err.Error(), "experimental app-server method \"memory/reset\" requires ClientCapabilities.ExperimentalAPI") {
		t.Fatalf("experimental memory/reset error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodMemoryReset) != nil {
		t.Fatal("memory/reset was sent after experimental method guard failure")
	}
}

func TestMemoryProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("memory-malformed-response"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Memory().Reset(context.Background())
	if err == nil {
		t.Fatal("Memory().Reset accepted malformed memory/reset response")
	}
	if !strings.Contains(err.Error(), "decode memory/reset response") ||
		!strings.Contains(err.Error(), `MemoryResetResponse: unknown field "extra"`) {
		t.Fatalf("malformed memory/reset response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodMemoryReset) == nil {
		t.Fatal("memory/reset was not sent before malformed response decode")
	}
}

func TestMockProtocolFamilyFacadeSendsTypedMethodAndDecodesResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	response, err := root.Mock().ExperimentalMethod(context.Background(), protocolv2.MockExperimentalMethodParams{
		Value: protocolv2.Value("ping"),
	})
	if err != nil {
		t.Fatalf("Mock().ExperimentalMethod returned error after experimental opt-in: %v", err)
	}
	if response.Echoed == nil || response.Echoed.Value == nil || *response.Echoed.Value != "ping" {
		t.Fatalf("mock/experimentalMethod response = %#v", response)
	}

	params := firstRecord(readRecords(t, record), "recv", protocolv2.MethodMockExperimentalMethod)["params"].(map[string]any)
	if params["value"] != "ping" {
		t.Fatalf("mock/experimentalMethod params = %#v", params)
	}
}

func TestMockProtocolFamilyFacadeRejectsExperimentalMethodBeforeWriteUnlessOptedIn(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Mock().ExperimentalMethod(context.Background(), protocolv2.MockExperimentalMethodParams{
		Value: protocolv2.Value("ping"),
	})
	if err == nil {
		t.Fatal("Mock().ExperimentalMethod accepted experimental method without opt-in")
	}
	if !strings.Contains(err.Error(), "experimental app-server method \"mock/experimentalMethod\" requires ClientCapabilities.ExperimentalAPI") {
		t.Fatalf("experimental mock/experimentalMethod error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodMockExperimentalMethod) != nil {
		t.Fatal("mock/experimentalMethod was sent after experimental method guard failure")
	}
}

func TestMockProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("mock-malformed-response"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Mock().ExperimentalMethod(context.Background(), protocolv2.MockExperimentalMethodParams{
		Value: protocolv2.Value("ping"),
	})
	if err == nil {
		t.Fatal("Mock().ExperimentalMethod accepted malformed mock/experimentalMethod response")
	}
	if !strings.Contains(err.Error(), "decode mock/experimentalMethod response") ||
		!strings.Contains(err.Error(), `MockExperimentalMethodResponse: unknown field "extra"`) {
		t.Fatalf("malformed mock/experimentalMethod response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodMockExperimentalMethod) == nil {
		t.Fatal("mock/experimentalMethod was not sent before malformed response decode")
	}
}

func TestProcessesProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	processes := root.Processes()
	if _, err := processes.Spawn(context.Background(), protocolv2.ProcessSpawnParams{
		Command:            []string{"bash", "-lc", "echo ok"},
		CWD:                "/workspace/facade",
		Env:                protocolv2.Value(map[string]*protocolv2.Nullable[string]{"PATH": protocolv2.Value("/bin"), "REMOVE": protocolv2.Null[string]()}),
		OutputBytesCap:     protocolv2.Value(uint64(1024)),
		ProcessHandle:      "proc-1",
		Size:               protocolv2.Value(protocolv2.ProcessTerminalSize{Cols: 120, Rows: 40}),
		StreamStdin:        Bool(true),
		StreamStdoutStderr: Bool(true),
		TimeoutMS:          protocolv2.Null[int64](),
		Tty:                Bool(true),
	}); err != nil {
		t.Fatalf("Processes().Spawn returned error after experimental opt-in: %v", err)
	}
	if _, err := processes.WriteStdin(context.Background(), protocolv2.ProcessWriteStdinParams{
		CloseStdin:    Bool(true),
		DeltaBase64:   protocolv2.Value("aGVsbG8="),
		ProcessHandle: "proc-1",
	}); err != nil {
		t.Fatalf("Processes().WriteStdin returned error after experimental opt-in: %v", err)
	}
	if _, err := processes.ResizePTY(context.Background(), protocolv2.ProcessResizePtyParams{
		ProcessHandle: "proc-1",
		Size:          protocolv2.ProcessTerminalSize{Cols: 80, Rows: 24},
	}); err != nil {
		t.Fatalf("Processes().ResizePTY returned error after experimental opt-in: %v", err)
	}
	if _, err := processes.Kill(context.Background(), protocolv2.ProcessKillParams{ProcessHandle: "proc-1"}); err != nil {
		t.Fatalf("Processes().Kill returned error after experimental opt-in: %v", err)
	}

	records := readRecords(t, record)
	for _, method := range []string{
		protocolv2.MethodProcessSpawn,
		protocolv2.MethodProcessWriteStdin,
		protocolv2.MethodProcessResizePTY,
		protocolv2.MethodProcessKill,
	} {
		if firstRecord(records, "recv", method) == nil {
			t.Fatalf("missing process method %s in records %#v", method, records)
		}
	}
	spawnParams := firstRecord(records, "recv", protocolv2.MethodProcessSpawn)["params"].(map[string]any)
	spawnCommand := spawnParams["command"].([]any)
	spawnSize := spawnParams["size"].(map[string]any)
	spawnEnv := spawnParams["env"].(map[string]any)
	if spawnParams["processHandle"] != "proc-1" || spawnParams["cwd"] != "/workspace/facade" ||
		len(spawnCommand) != 3 || spawnCommand[0] != "bash" || spawnParams["outputBytesCap"] != float64(1024) ||
		spawnSize["cols"] != float64(120) || spawnSize["rows"] != float64(40) ||
		spawnEnv["PATH"] != "/bin" || spawnEnv["REMOVE"] != nil ||
		spawnParams["streamStdin"] != true || spawnParams["streamStdoutStderr"] != true ||
		spawnParams["timeoutMs"] != nil || spawnParams["tty"] != true {
		t.Fatalf("process/spawn params = %#v", spawnParams)
	}
	writeParams := firstRecord(records, "recv", protocolv2.MethodProcessWriteStdin)["params"].(map[string]any)
	if writeParams["processHandle"] != "proc-1" || writeParams["deltaBase64"] != "aGVsbG8=" || writeParams["closeStdin"] != true {
		t.Fatalf("process/writeStdin params = %#v", writeParams)
	}
	resizeParams := firstRecord(records, "recv", protocolv2.MethodProcessResizePTY)["params"].(map[string]any)
	resizeSize := resizeParams["size"].(map[string]any)
	if resizeParams["processHandle"] != "proc-1" || resizeSize["cols"] != float64(80) || resizeSize["rows"] != float64(24) {
		t.Fatalf("process/resizePty params = %#v", resizeParams)
	}
	killParams := firstRecord(records, "recv", protocolv2.MethodProcessKill)["params"].(map[string]any)
	if killParams["processHandle"] != "proc-1" {
		t.Fatalf("process/kill params = %#v", killParams)
	}
}

func TestProcessesProtocolFamilyFacadeRejectsExperimentalMethodsBeforeWriteUnlessOptedIn(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	cases := []struct {
		name   string
		method string
		call   func() error
	}{
		{
			name:   "spawn",
			method: protocolv2.MethodProcessSpawn,
			call: func() error {
				_, err := root.Processes().Spawn(context.Background(), protocolv2.ProcessSpawnParams{
					Command:       []string{"sh"},
					CWD:           "/workspace/facade",
					ProcessHandle: "proc-1",
				})
				return err
			},
		},
		{
			name:   "writeStdin",
			method: protocolv2.MethodProcessWriteStdin,
			call: func() error {
				_, err := root.Processes().WriteStdin(context.Background(), protocolv2.ProcessWriteStdinParams{ProcessHandle: "proc-1"})
				return err
			},
		},
		{
			name:   "resizePty",
			method: protocolv2.MethodProcessResizePTY,
			call: func() error {
				_, err := root.Processes().ResizePTY(context.Background(), protocolv2.ProcessResizePtyParams{
					ProcessHandle: "proc-1",
					Size:          protocolv2.ProcessTerminalSize{Cols: 80, Rows: 24},
				})
				return err
			},
		},
		{
			name:   "kill",
			method: protocolv2.MethodProcessKill,
			call: func() error {
				_, err := root.Processes().Kill(context.Background(), protocolv2.ProcessKillParams{ProcessHandle: "proc-1"})
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatalf("%s accepted experimental method without opt-in", tc.method)
			}
			if !strings.Contains(err.Error(), "experimental app-server method \""+tc.method+"\" requires ClientCapabilities.ExperimentalAPI") {
				t.Fatalf("experimental %s error = %v", tc.method, err)
			}
		})
	}
	records := readRecords(t, record)
	for _, method := range []string{
		protocolv2.MethodProcessSpawn,
		protocolv2.MethodProcessWriteStdin,
		protocolv2.MethodProcessResizePTY,
		protocolv2.MethodProcessKill,
	} {
		if firstRecord(records, "recv", method) != nil {
			t.Fatalf("%s was sent after experimental method guard failure", method)
		}
	}
}

func TestProcessesProtocolFamilyFacadePreflightFailureDoesNotWriteRequest(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Processes().Spawn(context.Background(), protocolv2.ProcessSpawnParams{
		CWD:           "/workspace/facade",
		ProcessHandle: "proc-1",
	})
	if err == nil {
		t.Fatal("Processes().Spawn succeeded with nil required command")
	}
	if !strings.Contains(err.Error(), "encode process/spawn params") ||
		!strings.Contains(err.Error(), "ProcessSpawnParams.command: nil is not allowed") {
		t.Fatalf("process/spawn preflight error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodProcessSpawn) != nil {
		t.Fatal("process/spawn was sent after preflight failure")
	}
}

func TestProcessesProtocolFamilyFacadeRejectsMalformedTypedResponses(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		call    func(Processes) error
		wantSub string
	}{
		{
			name:   "spawn",
			method: protocolv2.MethodProcessSpawn,
			call: func(processes Processes) error {
				_, err := processes.Spawn(context.Background(), protocolv2.ProcessSpawnParams{
					Command:       []string{"sh"},
					CWD:           "/workspace/facade",
					ProcessHandle: "proc-1",
				})
				return err
			},
			wantSub: `ProcessSpawnResponse: unknown field "extra"`,
		},
		{
			name:   "writeStdin",
			method: protocolv2.MethodProcessWriteStdin,
			call: func(processes Processes) error {
				_, err := processes.WriteStdin(context.Background(), protocolv2.ProcessWriteStdinParams{ProcessHandle: "proc-1"})
				return err
			},
			wantSub: `ProcessWriteStdinResponse: unknown field "extra"`,
		},
		{
			name:   "resizePty",
			method: protocolv2.MethodProcessResizePTY,
			call: func(processes Processes) error {
				_, err := processes.ResizePTY(context.Background(), protocolv2.ProcessResizePtyParams{
					ProcessHandle: "proc-1",
					Size:          protocolv2.ProcessTerminalSize{Cols: 80, Rows: 24},
				})
				return err
			},
			wantSub: `ProcessResizePtyResponse: unknown field "extra"`,
		},
		{
			name:   "kill",
			method: protocolv2.MethodProcessKill,
			call: func(processes Processes) error {
				_, err := processes.Kill(context.Background(), protocolv2.ProcessKillParams{ProcessHandle: "proc-1"})
				return err
			},
			wantSub: `ProcessKillResponse: unknown field "extra"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := tempRecord(t)
			t.Setenv("CODEXSDK_FAKE_RECORD", record)
			root, err := New(ClientOptions{
				CWD:          t.TempDir(),
				Command:      fakeCommand("process-malformed-response"),
				Capabilities: ClientCapabilities{ExperimentalAPI: true},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()

			err = tc.call(root.Processes())
			if err == nil {
				t.Fatalf("%s accepted malformed response", tc.method)
			}
			if !strings.Contains(err.Error(), "decode "+tc.method+" response") ||
				!strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("malformed %s response error = %v", tc.method, err)
			}
			if firstRecord(readRecords(t, record), "recv", tc.method) == nil {
				t.Fatalf("%s was not sent before malformed response decode", tc.method)
			}
		})
	}
}

func TestExternalAgentConfigsProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	externalAgentConfigs := root.ExternalAgentConfigs()

	detect, err := externalAgentConfigs.Detect(context.Background(), protocolv2.ExternalAgentConfigDetectParams{
		CWDs:        protocolv2.Value([]string{"/repo"}),
		IncludeHome: Bool(true),
	})
	if err != nil {
		t.Fatalf("ExternalAgentConfigs().Detect returned error: %v", err)
	}
	if len(detect.Items) != 1 || detect.Items[0].ItemType != protocolv2.ExternalAgentConfigMigrationItemTypeCOMMANDS ||
		detect.Items[0].Details == nil || detect.Items[0].Details.Value == nil ||
		detect.Items[0].Details.Value.Commands == nil || len(*detect.Items[0].Details.Value.Commands) != 1 {
		t.Fatalf("externalAgentConfig/detect response = %#v", detect)
	}
	records := readRecords(t, record)
	detectParams := firstRecord(records, "recv", protocolv2.MethodExternalAgentConfigDetect)["params"].(map[string]any)
	cwds := detectParams["cwds"].([]any)
	if len(cwds) != 1 || cwds[0] != "/repo" || detectParams["includeHome"] != true {
		t.Fatalf("externalAgentConfig/detect params = %#v", detectParams)
	}
}

func TestExternalAgentConfigsProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("external-agent-configs-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.ExternalAgentConfigs().Detect(context.Background(), protocolv2.ExternalAgentConfigDetectParams{})
	if err == nil {
		t.Fatal("ExternalAgentConfigs().Detect accepted malformed externalAgentConfig/detect response")
	}
	if !strings.Contains(err.Error(), "decode externalAgentConfig/detect response") ||
		!strings.Contains(err.Error(), "ExternalAgentConfigDetectResponse.items") {
		t.Fatalf("malformed externalAgentConfig/detect response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodExternalAgentConfigDetect) == nil {
		t.Fatal("externalAgentConfig/detect was not sent before malformed response decode")
	}
}

func TestFeedbackProtocolFamilyFacadeSendsTypedMethodAndDecodesResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	response, err := root.Feedback().Upload(context.Background(), protocolv2.FeedbackUploadParams{
		Classification: "bug",
		ExtraLogFiles:  protocolv2.Value([]string{"logs/extra.txt"}),
		IncludeLogs:    Bool(false),
		Reason:         protocolv2.Null[string](),
		Tags:           protocolv2.Value(map[string]string{"area": "sdk"}),
		ThreadID:       protocolv2.Value("thread-1"),
	})
	if err != nil {
		t.Fatalf("Feedback().Upload returned error: %v", err)
	}
	if response.ThreadID != "thread-1" {
		t.Fatalf("feedback/upload response = %#v", response)
	}

	records := readRecords(t, record)
	params := firstRecord(records, "recv", protocolv2.MethodFeedbackUpload)["params"].(map[string]any)
	extraLogFiles := params["extraLogFiles"].([]any)
	tags := params["tags"].(map[string]any)
	if params["classification"] != "bug" || params["includeLogs"] != false ||
		len(extraLogFiles) != 1 || extraLogFiles[0] != "logs/extra.txt" ||
		params["reason"] != nil || tags["area"] != "sdk" || params["threadId"] != "thread-1" {
		t.Fatalf("feedback/upload params = %#v", params)
	}
}

func TestFeedbackProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("feedback-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Feedback().Upload(context.Background(), protocolv2.FeedbackUploadParams{
		Classification: "bug",
		IncludeLogs:    Bool(true),
	})
	if err == nil {
		t.Fatal("Feedback().Upload accepted malformed feedback/upload response")
	}
	if !strings.Contains(err.Error(), "decode feedback/upload response") ||
		!strings.Contains(err.Error(), "FeedbackUploadResponse.threadId") {
		t.Fatalf("malformed feedback/upload response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodFeedbackUpload) == nil {
		t.Fatal("feedback/upload was not sent before malformed response decode")
	}
}

func TestFuzzyFileSearchProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	fuzzyFileSearch := root.FuzzyFileSearch()

	search, err := fuzzyFileSearch.Search(context.Background(), protocolv2.FuzzyFileSearchParams{
		CancellationToken: protocolv2.Value("cancel-1"),
		Query:             "readme",
		Roots:             []string{"/repo"},
	})
	if err != nil {
		t.Fatalf("FuzzyFileSearch().Search returned error: %v", err)
	}
	if len(search.Files) != 1 || search.Files[0].FileName != "README.md" ||
		search.Files[0].MatchType != protocolv2.FuzzyFileSearchMatchTypeFile ||
		search.Files[0].Indices == nil || search.Files[0].Indices.Value == nil ||
		len(*search.Files[0].Indices.Value) != 2 {
		t.Fatalf("fuzzyFileSearch response = %#v", search)
	}

	records := readRecords(t, record)
	searchParams := firstRecord(records, "recv", protocolv2.MethodFuzzyFileSearch)["params"].(map[string]any)
	searchRoots := searchParams["roots"].([]any)
	if searchParams["cancellationToken"] != "cancel-1" || searchParams["query"] != "readme" ||
		len(searchRoots) != 1 || searchRoots[0] != "/repo" {
		t.Fatalf("fuzzyFileSearch params = %#v", searchParams)
	}
	_ = root.Close()

	record = tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err = New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	fuzzyFileSearch = root.FuzzyFileSearch()

	if _, err := fuzzyFileSearch.SessionStart(context.Background(), protocolv2.FuzzyFileSearchSessionStartParams{
		Roots:     []string{"/repo"},
		SessionID: "session-1",
	}); err != nil {
		t.Fatalf("FuzzyFileSearch().SessionStart returned error after experimental opt-in: %v", err)
	}
	if _, err := fuzzyFileSearch.SessionUpdate(context.Background(), protocolv2.FuzzyFileSearchSessionUpdateParams{
		Query:     "main",
		SessionID: "session-1",
	}); err != nil {
		t.Fatalf("FuzzyFileSearch().SessionUpdate returned error after experimental opt-in: %v", err)
	}
	if _, err := fuzzyFileSearch.SessionStop(context.Background(), protocolv2.FuzzyFileSearchSessionStopParams{
		SessionID: "session-1",
	}); err != nil {
		t.Fatalf("FuzzyFileSearch().SessionStop returned error after experimental opt-in: %v", err)
	}

	records = readRecords(t, record)
	startParams := firstRecord(records, "recv", protocolv2.MethodFuzzyFileSearchSessionStart)["params"].(map[string]any)
	startRoots := startParams["roots"].([]any)
	if len(startRoots) != 1 || startRoots[0] != "/repo" || startParams["sessionId"] != "session-1" {
		t.Fatalf("fuzzyFileSearch/sessionStart params = %#v", startParams)
	}
	updateParams := firstRecord(records, "recv", protocolv2.MethodFuzzyFileSearchSessionUpdate)["params"].(map[string]any)
	if updateParams["query"] != "main" || updateParams["sessionId"] != "session-1" {
		t.Fatalf("fuzzyFileSearch/sessionUpdate params = %#v", updateParams)
	}
	stopParams := firstRecord(records, "recv", protocolv2.MethodFuzzyFileSearchSessionStop)["params"].(map[string]any)
	if stopParams["sessionId"] != "session-1" {
		t.Fatalf("fuzzyFileSearch/sessionStop params = %#v", stopParams)
	}
}

func TestFuzzyFileSearchProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("fuzzy-file-search-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.FuzzyFileSearch().Search(context.Background(), protocolv2.FuzzyFileSearchParams{
		Query: "readme",
		Roots: []string{"/repo"},
	})
	if err == nil {
		t.Fatal("FuzzyFileSearch().Search accepted malformed fuzzyFileSearch response")
	}
	if !strings.Contains(err.Error(), "decode fuzzyFileSearch response") ||
		!strings.Contains(err.Error(), "FuzzyFileSearchResponse.files") {
		t.Fatalf("malformed fuzzyFileSearch response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodFuzzyFileSearch) == nil {
		t.Fatal("fuzzyFileSearch was not sent before malformed response decode")
	}
}

func TestFuzzyFileSearchProtocolFamilyFacadeRejectsExperimentalSessionBeforeWriteUnlessOptedIn(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.FuzzyFileSearch().SessionStart(context.Background(), protocolv2.FuzzyFileSearchSessionStartParams{
		Roots:     []string{"/repo"},
		SessionID: "session-1",
	})
	if err == nil {
		t.Fatal("FuzzyFileSearch().SessionStart accepted experimental method without opt-in")
	}
	if !strings.Contains(err.Error(), "experimental app-server method \"fuzzyFileSearch/sessionStart\" requires ClientCapabilities.ExperimentalAPI") {
		t.Fatalf("experimental fuzzyFileSearch/sessionStart error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodFuzzyFileSearchSessionStart) != nil {
		t.Fatal("fuzzyFileSearch/sessionStart was sent after experimental method guard failure")
	}
}

func TestHooksProtocolFamilyFacadeSendsTypedMethodAndDecodesResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	response, err := root.Hooks().List(context.Background(), protocolv2.HooksListParams{
		CWDs: &[]string{"/repo"},
	})
	if err != nil {
		t.Fatalf("Hooks().List returned error: %v", err)
	}
	if len(response.Data) != 1 || response.Data[0].CWD != "/repo" ||
		len(response.Data[0].Hooks) != 1 ||
		response.Data[0].Hooks[0].EventName != protocolv2.HookEventNamePreToolUse ||
		response.Data[0].Hooks[0].Command == nil ||
		response.Data[0].Hooks[0].Command.Value != nil {
		t.Fatalf("hooks/list response = %#v", response)
	}

	records := readRecords(t, record)
	params := firstRecord(records, "recv", protocolv2.MethodHooksList)["params"].(map[string]any)
	cwds := params["cwds"].([]any)
	if len(cwds) != 1 || cwds[0] != "/repo" {
		t.Fatalf("hooks/list params = %#v", params)
	}
}

func TestHooksProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("hooks-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Hooks().List(context.Background(), protocolv2.HooksListParams{})
	if err == nil {
		t.Fatal("Hooks().List accepted malformed hooks/list response")
	}
	if !strings.Contains(err.Error(), "decode hooks/list response") ||
		!strings.Contains(err.Error(), "HooksListResponse.data") {
		t.Fatalf("malformed hooks/list response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodHooksList) == nil {
		t.Fatal("hooks/list was not sent before malformed response decode")
	}
}

func TestReviewAndSkillsProtocolFamilyFacadesSendTypedMethodsAndDecodeResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	review, err := root.Reviews().Start(context.Background(), protocolv2.ReviewStartParams{
		Delivery: protocolv2.Value(protocolv2.ReviewDeliveryDetached),
		Target: protocolv2.NewReviewTargetBaseBranch(protocolv2.ReviewTargetBaseBranch{
			Branch: "main",
		}),
		ThreadID: "thread-1",
	})
	if err != nil {
		t.Fatalf("Reviews().Start returned error: %v", err)
	}
	if review.ReviewThreadID != "review-thread-1" || review.Turn.ID != "turn-review-1" ||
		review.Turn.Status != protocolv2.TurnStatusInProgress {
		t.Fatalf("review/start response = %#v", review)
	}

	forceReload := true
	skills, err := root.Skills().List(context.Background(), protocolv2.SkillsListParams{
		CWDs:        &[]string{"/repo"},
		ForceReload: &forceReload,
	})
	if err != nil {
		t.Fatalf("Skills().List returned error: %v", err)
	}
	if len(skills.Data) != 1 || skills.Data[0].CWD != "/repo" ||
		len(skills.Data[0].Skills) != 1 ||
		skills.Data[0].Skills[0].Scope != protocolv2.SkillScopeRepo ||
		skills.Data[0].Skills[0].Dependencies == nil ||
		skills.Data[0].Skills[0].Dependencies.Value == nil ||
		len(skills.Data[0].Skills[0].Dependencies.Value.Tools) != 1 {
		t.Fatalf("skills/list response = %#v", skills)
	}

	records := readRecords(t, record)
	reviewParams := firstRecord(records, "recv", protocolv2.MethodReviewStart)["params"].(map[string]any)
	reviewTarget := reviewParams["target"].(map[string]any)
	if reviewParams["delivery"] != "detached" || reviewParams["threadId"] != "thread-1" ||
		reviewTarget["type"] != "baseBranch" || reviewTarget["branch"] != "main" {
		t.Fatalf("review/start params = %#v", reviewParams)
	}
	skillsParams := firstRecord(records, "recv", protocolv2.MethodSkillsList)["params"].(map[string]any)
	cwds := skillsParams["cwds"].([]any)
	if len(cwds) != 1 || cwds[0] != "/repo" || skillsParams["forceReload"] != true {
		t.Fatalf("skills/list params = %#v", skillsParams)
	}
}

func TestReviewAndSkillsProtocolFamilyFacadesRejectMalformedTypedResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("review-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}

	_, err = root.Reviews().Start(context.Background(), protocolv2.ReviewStartParams{
		Target:   protocolv2.NewReviewTargetUncommittedChanges(),
		ThreadID: "thread-1",
	})
	if err == nil {
		t.Fatal("Reviews().Start accepted malformed review/start response")
	}
	if !strings.Contains(err.Error(), "decode review/start response") ||
		!strings.Contains(err.Error(), "ReviewStartResponse.turn") {
		t.Fatalf("malformed review/start response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodReviewStart) == nil {
		t.Fatal("review/start was not sent before malformed response decode")
	}
	_ = root.Close()

	record = tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err = New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("skills-list-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Skills().List(context.Background(), protocolv2.SkillsListParams{})
	if err == nil {
		t.Fatal("Skills().List accepted malformed skills/list response")
	}
	if !strings.Contains(err.Error(), "decode skills/list response") ||
		!strings.Contains(err.Error(), "SkillsListResponse.data") {
		t.Fatalf("malformed skills/list response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodSkillsList) == nil {
		t.Fatal("skills/list was not sent before malformed response decode")
	}
}

func TestMarketplaceProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	marketplace := root.Marketplace()

	add, err := marketplace.Add(context.Background(), protocolv2.MarketplaceAddParams{
		RefName:     protocolv2.Value("local"),
		Source:      "github:org/repo",
		SparsePaths: protocolv2.Value([]string{"plugins/a"}),
	})
	if err != nil {
		t.Fatalf("Marketplace().Add returned error: %v", err)
	}
	if add.MarketplaceName != "local" || add.InstalledRoot != "/codex/marketplaces/local" || add.AlreadyAdded {
		t.Fatalf("marketplace/add response = %#v", add)
	}
	remove, err := marketplace.Remove(context.Background(), protocolv2.MarketplaceRemoveParams{
		MarketplaceName: "local",
	})
	if err != nil {
		t.Fatalf("Marketplace().Remove returned error: %v", err)
	}
	if remove.MarketplaceName != "local" || remove.InstalledRoot == nil ||
		remove.InstalledRoot.Value == nil || *remove.InstalledRoot.Value != "/codex/marketplaces/local" {
		t.Fatalf("marketplace/remove response = %#v", remove)
	}
	upgrade, err := marketplace.Upgrade(context.Background(), protocolv2.MarketplaceUpgradeParams{
		MarketplaceName: protocolv2.Value("local"),
	})
	if err != nil {
		t.Fatalf("Marketplace().Upgrade returned error: %v", err)
	}
	if len(upgrade.SelectedMarketplaces) != 1 || upgrade.SelectedMarketplaces[0] != "local" ||
		len(upgrade.UpgradedRoots) != 1 || upgrade.UpgradedRoots[0] != "/codex/marketplaces/local" ||
		len(upgrade.Errors) != 0 {
		t.Fatalf("marketplace/upgrade response = %#v", upgrade)
	}

	records := readRecords(t, record)
	addParams := firstRecord(records, "recv", protocolv2.MethodMarketplaceAdd)["params"].(map[string]any)
	addSparsePaths := addParams["sparsePaths"].([]any)
	if addParams["refName"] != "local" || addParams["source"] != "github:org/repo" ||
		len(addSparsePaths) != 1 || addSparsePaths[0] != "plugins/a" {
		t.Fatalf("marketplace/add params = %#v", addParams)
	}
	removeParams := firstRecord(records, "recv", protocolv2.MethodMarketplaceRemove)["params"].(map[string]any)
	if removeParams["marketplaceName"] != "local" {
		t.Fatalf("marketplace/remove params = %#v", removeParams)
	}
	upgradeParams := firstRecord(records, "recv", protocolv2.MethodMarketplaceUpgrade)["params"].(map[string]any)
	if upgradeParams["marketplaceName"] != "local" {
		t.Fatalf("marketplace/upgrade params = %#v", upgradeParams)
	}
}

func TestMarketplaceProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("marketplace-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Marketplace().Upgrade(context.Background(), protocolv2.MarketplaceUpgradeParams{
		MarketplaceName: protocolv2.Value("local"),
	})
	if err == nil {
		t.Fatal("Marketplace().Upgrade accepted malformed marketplace/upgrade response")
	}
	if !strings.Contains(err.Error(), "decode marketplace/upgrade response") ||
		!strings.Contains(err.Error(), "MarketplaceUpgradeResponse.errors") {
		t.Fatalf("malformed marketplace/upgrade response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodMarketplaceUpgrade) == nil {
		t.Fatal("marketplace/upgrade was not sent before malformed response decode")
	}
}

func TestPluginsProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	plugins := root.Plugins()
	install, err := plugins.Install(context.Background(), protocolv2.PluginInstallParams{
		MarketplacePath:       protocolv2.Value("/marketplaces/local"),
		PluginName:            "plugin-one",
		RemoteMarketplaceName: protocolv2.Null[string](),
	})
	if err != nil {
		t.Fatalf("Plugins().Install returned error: %v", err)
	}
	if install.AuthPolicy != protocolv2.PluginAuthPolicyONINSTALL || len(install.AppsNeedingAuth) != 1 {
		t.Fatalf("plugin/install response = %#v", install)
	}
	list, err := plugins.List(context.Background(), protocolv2.PluginListParams{
		CWDs:             protocolv2.Value([]string{"/repo"}),
		MarketplaceKinds: protocolv2.Value([]protocolv2.PluginListMarketplaceKind{protocolv2.PluginListMarketplaceKindLocal}),
	})
	if err != nil {
		t.Fatalf("Plugins().List returned error: %v", err)
	}
	if len(list.Marketplaces) != 1 || list.Marketplaces[0].Name != "market" || len(list.Marketplaces[0].Plugins) != 1 {
		t.Fatalf("plugin/list response = %#v", list)
	}
	read, err := plugins.Read(context.Background(), protocolv2.PluginReadParams{
		PluginName: "plugin-one",
	})
	if err != nil {
		t.Fatalf("Plugins().Read returned error: %v", err)
	}
	if read.Plugin.MarketplaceName != "market" || read.Plugin.Summary.ID != "plugin-1" {
		t.Fatalf("plugin/read response = %#v", read)
	}
	if _, err := plugins.ShareDelete(context.Background(), protocolv2.PluginShareDeleteParams{RemotePluginID: "remote-1"}); err != nil {
		t.Fatalf("Plugins().ShareDelete returned error: %v", err)
	}
	shareList, err := plugins.ShareList(context.Background(), protocolv2.PluginShareListParams{})
	if err != nil {
		t.Fatalf("Plugins().ShareList returned error: %v", err)
	}
	if len(shareList.Data) != 1 || shareList.Data[0].LocalPluginPath == nil ||
		shareList.Data[0].LocalPluginPath.Value == nil || *shareList.Data[0].LocalPluginPath.Value != "/plugins/plugin-one" {
		t.Fatalf("plugin/share/list response = %#v", shareList)
	}
	shareSave, err := plugins.ShareSave(context.Background(), protocolv2.PluginShareSaveParams{
		Discoverability: protocolv2.Value(protocolv2.PluginShareDiscoverabilityPRIVATE),
		PluginPath:      "/plugins/plugin-one",
		RemotePluginID:  protocolv2.Null[string](),
		ShareTargets: protocolv2.Value([]protocolv2.PluginShareTarget{{
			PrincipalID:   "group-1",
			PrincipalType: protocolv2.PluginSharePrincipalTypeGroup,
			Role:          protocolv2.PluginShareTargetRoleReader,
		}}),
	})
	if err != nil {
		t.Fatalf("Plugins().ShareSave returned error: %v", err)
	}
	if shareSave.RemotePluginID != "remote-1" || shareSave.ShareURL != "https://example.test/plugin" {
		t.Fatalf("plugin/share/save response = %#v", shareSave)
	}
	updateTargets, err := plugins.ShareUpdateTargets(context.Background(), protocolv2.PluginShareUpdateTargetsParams{
		Discoverability: protocolv2.PluginShareUpdateDiscoverabilityPRIVATE,
		RemotePluginID:  "remote-1",
		ShareTargets: []protocolv2.PluginShareTarget{{
			PrincipalID:   "user-1",
			PrincipalType: protocolv2.PluginSharePrincipalTypeUser,
			Role:          protocolv2.PluginShareTargetRoleEditor,
		}},
	})
	if err != nil {
		t.Fatalf("Plugins().ShareUpdateTargets returned error: %v", err)
	}
	if updateTargets.Discoverability != protocolv2.PluginShareDiscoverabilityPRIVATE || len(updateTargets.Principals) != 1 {
		t.Fatalf("plugin/share/updateTargets response = %#v", updateTargets)
	}
	skill, err := plugins.SkillRead(context.Background(), protocolv2.PluginSkillReadParams{
		RemoteMarketplaceName: "market",
		RemotePluginID:        "remote-1",
		SkillName:             "review",
	})
	if err != nil {
		t.Fatalf("Plugins().SkillRead returned error: %v", err)
	}
	if skill.Contents == nil || skill.Contents.Value == nil || *skill.Contents.Value != "skill contents" {
		t.Fatalf("plugin/skill/read response = %#v", skill)
	}
	if _, err := plugins.Uninstall(context.Background(), protocolv2.PluginUninstallParams{PluginID: "plugin-1"}); err != nil {
		t.Fatalf("Plugins().Uninstall returned error: %v", err)
	}

	records := readRecords(t, record)
	for _, method := range []string{
		protocolv2.MethodPluginInstall,
		protocolv2.MethodPluginList,
		protocolv2.MethodPluginRead,
		protocolv2.MethodPluginShareDelete,
		protocolv2.MethodPluginShareList,
		protocolv2.MethodPluginShareSave,
		protocolv2.MethodPluginShareUpdateTargets,
		protocolv2.MethodPluginSkillRead,
		protocolv2.MethodPluginUninstall,
	} {
		if firstRecord(records, "recv", method) == nil {
			t.Fatalf("missing plugin facade method %s in records %#v", method, records)
		}
	}
	installParams := firstRecord(records, "recv", protocolv2.MethodPluginInstall)["params"].(map[string]any)
	if installParams["marketplacePath"] != "/marketplaces/local" || installParams["pluginName"] != "plugin-one" ||
		installParams["remoteMarketplaceName"] != nil {
		t.Fatalf("plugin/install params = %#v", installParams)
	}
	shareSaveParams := firstRecord(records, "recv", protocolv2.MethodPluginShareSave)["params"].(map[string]any)
	shareTargets := shareSaveParams["shareTargets"].([]any)
	if shareSaveParams["discoverability"] != "PRIVATE" || shareSaveParams["pluginPath"] != "/plugins/plugin-one" ||
		shareSaveParams["remotePluginId"] != nil || len(shareTargets) != 1 {
		t.Fatalf("plugin/share/save params = %#v", shareSaveParams)
	}
	updateTargetsParams := firstRecord(records, "recv", protocolv2.MethodPluginShareUpdateTargets)["params"].(map[string]any)
	if updateTargetsParams["discoverability"] != "PRIVATE" || updateTargetsParams["remotePluginId"] != "remote-1" {
		t.Fatalf("plugin/share/updateTargets params = %#v", updateTargetsParams)
	}
}

func TestPluginsProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("plugins-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Plugins().List(context.Background(), protocolv2.PluginListParams{})
	if err == nil {
		t.Fatal("Plugins().List accepted malformed plugin/list response")
	}
	if !strings.Contains(err.Error(), "decode plugin/list response") ||
		!strings.Contains(err.Error(), "PluginListResponse.marketplaces") {
		t.Fatalf("malformed plugin/list response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodPluginList) == nil {
		t.Fatal("plugin/list was not sent before malformed response decode")
	}
}

func TestMCPProtocolFamilyFacadesSendTypedMethodsAndDecodeResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	mcpServers := root.MCPServers()
	oauth, err := mcpServers.OAuthLogin(context.Background(), protocolv2.McpServerOauthLoginParams{
		Name:        "server-1",
		Scopes:      protocolv2.Value([]string{"repo"}),
		TimeoutSecs: protocolv2.Value(int64(30)),
	})
	if err != nil {
		t.Fatalf("MCPServers().OAuthLogin returned error: %v", err)
	}
	if oauth.AuthorizationURL != "https://example.test/oauth" {
		t.Fatalf("mcpServer/oauth/login response = %#v", oauth)
	}
	resource, err := mcpServers.ResourceRead(context.Background(), protocolv2.McpResourceReadParams{
		Server:   "server-1",
		ThreadID: protocolv2.Value("thread-1"),
		URI:      "file://README.md",
	})
	if err != nil {
		t.Fatalf("MCPServers().ResourceRead returned error: %v", err)
	}
	if len(resource.Contents) != 1 || resource.Contents[0].Kind() != protocolv2.ResourceContentKindText {
		t.Fatalf("mcpServer/resource/read response = %#v", resource)
	}
	text, ok := resource.Contents[0].AsText()
	if !ok || text.Text != "hello" || text.URI != "file://README.md" {
		t.Fatalf("mcpServer/resource/read text content = %#v, ok=%v", text, ok)
	}
	arguments := protocolv2.JSONObject(map[string]protocolv2.JSONValue{"query": protocolv2.JSONString("docs")})
	tool, err := mcpServers.ToolCall(context.Background(), protocolv2.McpServerToolCallParams{
		Arguments: &arguments,
		Server:    "server-1",
		ThreadID:  "thread-1",
		Tool:      "search",
	})
	if err != nil {
		t.Fatalf("MCPServers().ToolCall returned error: %v", err)
	}
	if len(tool.Content) != 1 || tool.IsError == nil || tool.IsError.Value == nil || *tool.IsError.Value {
		t.Fatalf("mcpServer/tool/call response = %#v", tool)
	}
	status, err := root.MCPServerStatus().List(context.Background(), protocolv2.ListMcpServerStatusParams{
		Detail: protocolv2.Value(protocolv2.McpServerStatusDetailToolsAndAuthOnly),
		Limit:  protocolv2.Value(uint32(10)),
	})
	if err != nil {
		t.Fatalf("MCPServerStatus().List returned error: %v", err)
	}
	if len(status.Data) != 1 || status.Data[0].Name != "server-1" ||
		status.Data[0].AuthStatus != protocolv2.McpAuthStatusOAuth ||
		len(status.Data[0].Tools) != 1 ||
		status.NextCursor == nil || status.NextCursor.Value == nil || *status.NextCursor.Value != "cursor-2" {
		t.Fatalf("mcpServerStatus/list response = %#v", status)
	}

	records := readRecords(t, record)
	oauthParams := firstRecord(records, "recv", protocolv2.MethodMCPServerOAuthLogin)["params"].(map[string]any)
	oauthScopes := oauthParams["scopes"].([]any)
	if oauthParams["name"] != "server-1" || len(oauthScopes) != 1 || oauthScopes[0] != "repo" ||
		oauthParams["timeoutSecs"] != float64(30) {
		t.Fatalf("mcpServer/oauth/login params = %#v", oauthParams)
	}
	resourceParams := firstRecord(records, "recv", protocolv2.MethodMCPServerResourceRead)["params"].(map[string]any)
	if resourceParams["server"] != "server-1" || resourceParams["threadId"] != "thread-1" ||
		resourceParams["uri"] != "file://README.md" {
		t.Fatalf("mcpServer/resource/read params = %#v", resourceParams)
	}
	toolParams := firstRecord(records, "recv", protocolv2.MethodMCPServerToolCall)["params"].(map[string]any)
	toolArguments := toolParams["arguments"].(map[string]any)
	if toolParams["server"] != "server-1" || toolParams["threadId"] != "thread-1" ||
		toolParams["tool"] != "search" || toolArguments["query"] != "docs" {
		t.Fatalf("mcpServer/tool/call params = %#v", toolParams)
	}
	statusParams := firstRecord(records, "recv", protocolv2.MethodMCPServerStatusList)["params"].(map[string]any)
	if statusParams["detail"] != "toolsAndAuthOnly" || statusParams["limit"] != float64(10) {
		t.Fatalf("mcpServerStatus/list params = %#v", statusParams)
	}
}

func TestMCPProtocolFamilyFacadesRejectMalformedTypedResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("mcp-servers-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = root.MCPServers().ResourceRead(context.Background(), protocolv2.McpResourceReadParams{
		Server: "server-1",
		URI:    "file://README.md",
	})
	if err == nil {
		t.Fatal("MCPServers().ResourceRead accepted malformed mcpServer/resource/read response")
	}
	if !strings.Contains(err.Error(), "decode mcpServer/resource/read response") ||
		!strings.Contains(err.Error(), "McpResourceReadResponse.contents") {
		t.Fatalf("malformed mcpServer/resource/read response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodMCPServerResourceRead) == nil {
		t.Fatal("mcpServer/resource/read was not sent before malformed response decode")
	}
	_ = root.Close()

	record = tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err = New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("mcp-server-status-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	_, err = root.MCPServerStatus().List(context.Background(), protocolv2.ListMcpServerStatusParams{})
	if err == nil {
		t.Fatal("MCPServerStatus().List accepted malformed mcpServerStatus/list response")
	}
	if !strings.Contains(err.Error(), "decode mcpServerStatus/list response") ||
		!strings.Contains(err.Error(), "ListMcpServerStatusResponse.data") {
		t.Fatalf("malformed mcpServerStatus/list response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodMCPServerStatusList) == nil {
		t.Fatal("mcpServerStatus/list was not sent before malformed response decode")
	}
}

func TestCommandProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	commands := root.Commands()

	exec, err := commands.Exec(context.Background(), protocolv2.CommandExecParams{
		Command:     []string{"bash", "-lc", "echo ok"},
		CWD:         protocolv2.Value("/workspace"),
		ProcessID:   protocolv2.Value("proc-1"),
		Size:        protocolv2.Value(protocolv2.CommandExecTerminalSize{Cols: 120, Rows: 40}),
		StreamStdin: Bool(true),
		Tty:         Bool(false),
	})
	if err != nil {
		t.Fatalf("Commands().Exec returned error: %v", err)
	}
	if exec.ExitCode != 0 || exec.Stdout != "ok\n" || exec.Stderr != "" {
		t.Fatalf("command/exec response = %#v", exec)
	}
	if _, err := commands.ExecResize(context.Background(), protocolv2.CommandExecResizeParams{
		ProcessID: "proc-1",
		Size:      protocolv2.CommandExecTerminalSize{Cols: 100, Rows: 30},
	}); err != nil {
		t.Fatalf("Commands().ExecResize returned error: %v", err)
	}
	if _, err := commands.ExecTerminate(context.Background(), protocolv2.CommandExecTerminateParams{ProcessID: "proc-1"}); err != nil {
		t.Fatalf("Commands().ExecTerminate returned error: %v", err)
	}
	if _, err := commands.ExecWrite(context.Background(), protocolv2.CommandExecWriteParams{
		CloseStdin:  Bool(false),
		DeltaBase64: protocolv2.Value("aW5wdXQK"),
		ProcessID:   "proc-1",
	}); err != nil {
		t.Fatalf("Commands().ExecWrite returned error: %v", err)
	}

	records := readRecords(t, record)
	wantMethods := []string{
		protocolv2.MethodCommandExec,
		protocolv2.MethodCommandExecResize,
		protocolv2.MethodCommandExecTerminate,
		protocolv2.MethodCommandExecWrite,
	}
	for _, method := range wantMethods {
		if firstRecord(records, "recv", method) == nil {
			t.Fatalf("missing command facade method %s in records %#v", method, records)
		}
	}
	execParams := firstRecord(records, "recv", protocolv2.MethodCommandExec)["params"].(map[string]any)
	if got := strings.Join(stringify(execParams["command"].([]any)), " "); got != "bash -lc echo ok" {
		t.Fatalf("command/exec command params = %#v", execParams["command"])
	}
	execSize := execParams["size"].(map[string]any)
	if execParams["cwd"] != "/workspace" || execParams["processId"] != "proc-1" ||
		execParams["streamStdin"] != true || execParams["tty"] != false ||
		execSize["cols"] != float64(120) || execSize["rows"] != float64(40) {
		t.Fatalf("command/exec params = %#v", execParams)
	}
	resizeParams := firstRecord(records, "recv", protocolv2.MethodCommandExecResize)["params"].(map[string]any)
	resizeSize := resizeParams["size"].(map[string]any)
	if resizeParams["processId"] != "proc-1" || resizeSize["cols"] != float64(100) || resizeSize["rows"] != float64(30) {
		t.Fatalf("command/exec/resize params = %#v", resizeParams)
	}
	terminateParams := firstRecord(records, "recv", protocolv2.MethodCommandExecTerminate)["params"].(map[string]any)
	if terminateParams["processId"] != "proc-1" {
		t.Fatalf("command/exec/terminate params = %#v", terminateParams)
	}
	writeParams := firstRecord(records, "recv", protocolv2.MethodCommandExecWrite)["params"].(map[string]any)
	if writeParams["processId"] != "proc-1" || writeParams["closeStdin"] != false || writeParams["deltaBase64"] != "aW5wdXQK" {
		t.Fatalf("command/exec/write params = %#v", writeParams)
	}
}

func TestCommandProtocolFamilyFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("command-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Commands().Exec(context.Background(), protocolv2.CommandExecParams{Command: []string{"echo", "ok"}})
	if err == nil {
		t.Fatal("Commands().Exec accepted malformed command/exec response")
	}
	if !strings.Contains(err.Error(), "decode command/exec response") ||
		!strings.Contains(err.Error(), "CommandExecResponse.stderr") {
		t.Fatalf("malformed command/exec response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodCommandExec) == nil {
		t.Fatal("command/exec was not sent before malformed response decode")
	}
}

func TestAccountProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	accounts := root.Accounts()

	cancel, err := accounts.LoginCancel(context.Background(), protocolv2.CancelLoginAccountParams{LoginID: "login-1"})
	if err != nil {
		t.Fatalf("Accounts().LoginCancel returned error: %v", err)
	}
	if cancel.Status != protocolv2.CancelLoginAccountStatusCanceled {
		t.Fatalf("account/login/cancel response = %#v", cancel)
	}
	login, err := accounts.LoginStart(context.Background(), protocolv2.NewLoginAccountParamsChatGPT(protocolv2.LoginAccountParamsChatGPT{
		CodexStreamlinedLogin: Bool(true),
	}))
	if err != nil {
		t.Fatalf("Accounts().LoginStart returned error: %v", err)
	}
	chatGPT, ok := login.AsChatGPT()
	if !ok || chatGPT.LoginID != "login-1" || chatGPT.AuthURL != "https://example.test/auth" {
		t.Fatalf("account/login/start response = %#v", login)
	}
	if _, err := accounts.Logout(context.Background()); err != nil {
		t.Fatalf("Accounts().Logout returned error: %v", err)
	}
	rateLimits, err := accounts.RateLimitsRead(context.Background())
	if err != nil {
		t.Fatalf("Accounts().RateLimitsRead returned error: %v", err)
	}
	if rateLimits.RateLimits.PlanType == nil || rateLimits.RateLimits.PlanType.Value == nil || *rateLimits.RateLimits.PlanType.Value != protocolv2.PlanTypePlus {
		t.Fatalf("account/rateLimits/read response = %#v", rateLimits)
	}
	account, err := accounts.Read(context.Background(), protocolv2.GetAccountParams{RefreshToken: Bool(true)})
	if err != nil {
		t.Fatalf("Accounts().Read returned error: %v", err)
	}
	if account.RequiresOpenaiAuth || account.Account == nil || account.Account.Value == nil {
		t.Fatalf("account/read response = %#v", account)
	}
	if account.Account.Value.Kind() != protocolv2.AccountKindAPIKey {
		t.Fatalf("account/read account variant = %#v", account.Account.Value)
	}
	nudge, err := accounts.SendAddCreditsNudgeEmail(context.Background(), protocolv2.SendAddCreditsNudgeEmailParams{
		CreditType: protocolv2.AddCreditsNudgeCreditTypeUsageLimit,
	})
	if err != nil {
		t.Fatalf("Accounts().SendAddCreditsNudgeEmail returned error: %v", err)
	}
	if nudge.Status != protocolv2.AddCreditsNudgeEmailStatusSent {
		t.Fatalf("account/sendAddCreditsNudgeEmail response = %#v", nudge)
	}

	records := readRecords(t, record)
	wantMethods := []string{
		protocolv2.MethodAccountLoginCancel,
		protocolv2.MethodAccountLoginStart,
		protocolv2.MethodAccountLogout,
		protocolv2.MethodAccountRateLimitsRead,
		protocolv2.MethodAccountRead,
		protocolv2.MethodAccountSendAddCreditsNudgeEmail,
	}
	for _, method := range wantMethods {
		if firstRecord(records, "recv", method) == nil {
			t.Fatalf("missing account facade method %s in records %#v", method, records)
		}
	}
	cancelParams := firstRecord(records, "recv", protocolv2.MethodAccountLoginCancel)["params"].(map[string]any)
	if cancelParams["loginId"] != "login-1" {
		t.Fatalf("account/login/cancel params = %#v", cancelParams)
	}
	loginParams := firstRecord(records, "recv", protocolv2.MethodAccountLoginStart)["params"].(map[string]any)
	if loginParams["type"] != "chatgpt" || loginParams["codexStreamlinedLogin"] != true {
		t.Fatalf("account/login/start params = %#v", loginParams)
	}
	if params := firstRecord(records, "recv", protocolv2.MethodAccountLogout)["params"]; params != nil {
		t.Fatalf("account/logout params = %#v, want omitted", params)
	}
	if params := firstRecord(records, "recv", protocolv2.MethodAccountRateLimitsRead)["params"]; params != nil {
		t.Fatalf("account/rateLimits/read params = %#v, want omitted", params)
	}
	readParams := firstRecord(records, "recv", protocolv2.MethodAccountRead)["params"].(map[string]any)
	if readParams["refreshToken"] != true {
		t.Fatalf("account/read params = %#v", readParams)
	}
	nudgeParams := firstRecord(records, "recv", protocolv2.MethodAccountSendAddCreditsNudgeEmail)["params"].(map[string]any)
	if nudgeParams["creditType"] != "usage_limit" {
		t.Fatalf("account/sendAddCreditsNudgeEmail params = %#v", nudgeParams)
	}
}

func TestFSProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	fs := root.FS()

	if _, err := fs.Copy(context.Background(), protocolv2.FsCopyParams{SourcePath: "/src", DestinationPath: "/dst", Recursive: Bool(true)}); err != nil {
		t.Fatalf("FS().Copy returned error: %v", err)
	}
	if _, err := fs.CreateDirectory(context.Background(), protocolv2.FsCreateDirectoryParams{Path: "/dir", Recursive: protocolv2.Value(true)}); err != nil {
		t.Fatalf("FS().CreateDirectory returned error: %v", err)
	}
	metadata, err := fs.GetMetadata(context.Background(), protocolv2.FsGetMetadataParams{Path: "/file.txt"})
	if err != nil {
		t.Fatalf("FS().GetMetadata returned error: %v", err)
	}
	if !metadata.IsFile || metadata.IsDirectory || metadata.ModifiedAtMS != 2000 {
		t.Fatalf("fs/getMetadata response = %#v", metadata)
	}
	directory, err := fs.ReadDirectory(context.Background(), protocolv2.FsReadDirectoryParams{Path: "/dir"})
	if err != nil {
		t.Fatalf("FS().ReadDirectory returned error: %v", err)
	}
	if len(directory.Entries) != 1 || directory.Entries[0].FileName != "file.txt" {
		t.Fatalf("fs/readDirectory response = %#v", directory)
	}
	file, err := fs.ReadFile(context.Background(), protocolv2.FsReadFileParams{Path: "/file.txt"})
	if err != nil {
		t.Fatalf("FS().ReadFile returned error: %v", err)
	}
	if file.DataBase64 != "ZmFrZQ==" {
		t.Fatalf("fs/readFile response = %#v", file)
	}
	if _, err := fs.Remove(context.Background(), protocolv2.FsRemoveParams{Path: "/old", Force: protocolv2.Value(true), Recursive: protocolv2.Value(false)}); err != nil {
		t.Fatalf("FS().Remove returned error: %v", err)
	}
	if _, err := fs.Unwatch(context.Background(), protocolv2.FsUnwatchParams{WatchID: "watch-1"}); err != nil {
		t.Fatalf("FS().Unwatch returned error: %v", err)
	}
	watch, err := fs.Watch(context.Background(), protocolv2.FsWatchParams{Path: "/watched", WatchID: "watch-2"})
	if err != nil {
		t.Fatalf("FS().Watch returned error: %v", err)
	}
	if watch.Path != "/watched" {
		t.Fatalf("fs/watch response = %#v", watch)
	}
	if _, err := fs.WriteFile(context.Background(), protocolv2.FsWriteFileParams{Path: "/file.txt", DataBase64: "ZmFrZQ=="}); err != nil {
		t.Fatalf("FS().WriteFile returned error: %v", err)
	}

	records := readRecords(t, record)
	wantMethods := []string{
		protocolv2.MethodFSCopy,
		protocolv2.MethodFSCreateDirectory,
		protocolv2.MethodFSGetMetadata,
		protocolv2.MethodFSReadDirectory,
		protocolv2.MethodFSReadFile,
		protocolv2.MethodFSRemove,
		protocolv2.MethodFSUnwatch,
		protocolv2.MethodFSWatch,
		protocolv2.MethodFSWriteFile,
	}
	for _, method := range wantMethods {
		if firstRecord(records, "recv", method) == nil {
			t.Fatalf("missing FS facade method %s in records %#v", method, records)
		}
	}
	copyParams := firstRecord(records, "recv", protocolv2.MethodFSCopy)["params"].(map[string]any)
	if copyParams["sourcePath"] != "/src" || copyParams["destinationPath"] != "/dst" || copyParams["recursive"] != true {
		t.Fatalf("fs/copy params = %#v", copyParams)
	}
	createParams := firstRecord(records, "recv", protocolv2.MethodFSCreateDirectory)["params"].(map[string]any)
	if createParams["path"] != "/dir" || createParams["recursive"] != true {
		t.Fatalf("fs/createDirectory params = %#v", createParams)
	}
	metadataParams := firstRecord(records, "recv", protocolv2.MethodFSGetMetadata)["params"].(map[string]any)
	if metadataParams["path"] != "/file.txt" {
		t.Fatalf("fs/getMetadata params = %#v", metadataParams)
	}
	readDirectoryParams := firstRecord(records, "recv", protocolv2.MethodFSReadDirectory)["params"].(map[string]any)
	if readDirectoryParams["path"] != "/dir" {
		t.Fatalf("fs/readDirectory params = %#v", readDirectoryParams)
	}
	readFileParams := firstRecord(records, "recv", protocolv2.MethodFSReadFile)["params"].(map[string]any)
	if readFileParams["path"] != "/file.txt" {
		t.Fatalf("fs/readFile params = %#v", readFileParams)
	}
	removeParams := firstRecord(records, "recv", protocolv2.MethodFSRemove)["params"].(map[string]any)
	if removeParams["path"] != "/old" || removeParams["force"] != true || removeParams["recursive"] != false {
		t.Fatalf("fs/remove params = %#v", removeParams)
	}
	unwatchParams := firstRecord(records, "recv", protocolv2.MethodFSUnwatch)["params"].(map[string]any)
	if unwatchParams["watchId"] != "watch-1" {
		t.Fatalf("fs/unwatch params = %#v", unwatchParams)
	}
	watchParams := firstRecord(records, "recv", protocolv2.MethodFSWatch)["params"].(map[string]any)
	if watchParams["path"] != "/watched" || watchParams["watchId"] != "watch-2" {
		t.Fatalf("fs/watch params = %#v", watchParams)
	}
	writeParams := firstRecord(records, "recv", protocolv2.MethodFSWriteFile)["params"].(map[string]any)
	if writeParams["path"] != "/file.txt" || writeParams["dataBase64"] != "ZmFrZQ==" {
		t.Fatalf("fs/writeFile params = %#v", writeParams)
	}
}

func TestAdditionalProtocolFamilyFacadesSendTypedMethodsAndDecodeResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	apps, err := root.Apps().List(context.Background(), protocolv2.AppsListParams{
		Limit:    protocolv2.Value(uint32(5)),
		ThreadID: protocolv2.Value("thread-1"),
	})
	if err != nil {
		t.Fatalf("Apps().List returned error: %v", err)
	}
	if len(apps.Data) != 1 || apps.Data[0].ID != "app-1" {
		t.Fatalf("app/list response = %#v", apps)
	}
	models, err := root.Models().List(context.Background(), protocolv2.ModelListParams{
		IncludeHidden: protocolv2.Value(true),
		Limit:         protocolv2.Value(uint32(10)),
	})
	if err != nil {
		t.Fatalf("Models().List returned error: %v", err)
	}
	if len(models.Data) != 0 || models.NextCursor == nil || models.NextCursor.Value == nil || *models.NextCursor.Value != "model-cursor" {
		t.Fatalf("model/list response = %#v", models)
	}
	capabilities, err := root.ModelProviders().CapabilitiesRead(context.Background(), protocolv2.ModelProviderCapabilitiesReadParams{})
	if err != nil {
		t.Fatalf("ModelProviders().CapabilitiesRead returned error: %v", err)
	}
	if !capabilities.WebSearch || !capabilities.NamespaceTools || capabilities.ImageGeneration {
		t.Fatalf("modelProvider/capabilities/read response = %#v", capabilities)
	}
	skills, err := root.Skills().ConfigWrite(context.Background(), protocolv2.SkillsConfigWriteParams{
		Enabled: true,
		Name:    protocolv2.Value("skill-a"),
	})
	if err != nil {
		t.Fatalf("Skills().ConfigWrite returned error: %v", err)
	}
	if !skills.EffectiveEnabled {
		t.Fatalf("skills/config/write response = %#v", skills)
	}
	readiness, err := root.WindowsSandbox().Readiness(context.Background())
	if err != nil {
		t.Fatalf("WindowsSandbox().Readiness returned error: %v", err)
	}
	if readiness.Status != protocolv2.WindowsSandboxReadinessReady {
		t.Fatalf("windowsSandbox/readiness response = %#v", readiness)
	}
	setup, err := root.WindowsSandbox().SetupStart(context.Background(), protocolv2.WindowsSandboxSetupStartParams{
		CWD:  protocolv2.Value("/workspace/facade"),
		Mode: protocolv2.WindowsSandboxSetupModeUnelevated,
	})
	if err != nil {
		t.Fatalf("WindowsSandbox().SetupStart returned error: %v", err)
	}
	if !setup.Started {
		t.Fatalf("windowsSandbox/setupStart response = %#v", setup)
	}

	records := readRecords(t, record)
	appList := firstRecord(records, "recv", protocolv2.MethodAppList)["params"].(map[string]any)
	if appList["limit"] != float64(5) || appList["threadId"] != "thread-1" {
		t.Fatalf("app/list params = %#v", appList)
	}
	modelList := firstRecord(records, "recv", protocolv2.MethodModelList)["params"].(map[string]any)
	if modelList["includeHidden"] != true || modelList["limit"] != float64(10) {
		t.Fatalf("model/list params = %#v", modelList)
	}
	if params := firstRecord(records, "recv", protocolv2.MethodModelProviderCapabilitiesRead)["params"].(map[string]any); len(params) != 0 {
		t.Fatalf("modelProvider/capabilities/read params = %#v, want empty object", params)
	}
	skillsWrite := firstRecord(records, "recv", protocolv2.MethodSkillsConfigWrite)["params"].(map[string]any)
	if skillsWrite["enabled"] != true || skillsWrite["name"] != "skill-a" {
		t.Fatalf("skills/config/write params = %#v", skillsWrite)
	}
	if params := firstRecord(records, "recv", protocolv2.MethodWindowsSandboxReadiness)["params"]; params != nil {
		t.Fatalf("windowsSandbox/readiness params = %#v, want omitted", params)
	}
	setupStart := firstRecord(records, "recv", protocolv2.MethodWindowsSandboxSetupStart)["params"].(map[string]any)
	if setupStart["cwd"] != "/workspace/facade" || setupStart["mode"] != "unelevated" {
		t.Fatalf("windowsSandbox/setupStart params = %#v", setupStart)
	}
}

func TestProtocolFamilyFacadesSendTypedMethodsAndDecodeResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	threads := root.Threads()
	turns := root.Turns()

	started, err := threads.Start(context.Background(), protocolv2.ThreadStartParams{
		CWD:   protocolv2.Value("/workspace/facade"),
		Model: protocolv2.Value("gpt-facade"),
	})
	if err != nil {
		t.Fatalf("Threads().Start returned error: %v", err)
	}
	if started.Thread.ID != "thread-1" || started.Model != "gpt-facade" {
		t.Fatalf("thread/start response = %#v", started)
	}
	resumed, err := threads.Resume(context.Background(), protocolv2.ThreadResumeParams{
		ThreadID: "thread-1",
		Model:    protocolv2.Value("gpt-resume"),
	})
	if err != nil {
		t.Fatalf("Threads().Resume returned error: %v", err)
	}
	if resumed.Thread.ID != "thread-1" || resumed.Model != "gpt-resume" {
		t.Fatalf("thread/resume response = %#v", resumed)
	}
	forked, err := threads.Fork(context.Background(), protocolv2.ThreadForkParams{
		ThreadID: "thread-1",
		Model:    protocolv2.Value("gpt-fork"),
	})
	if err != nil {
		t.Fatalf("Threads().Fork returned error: %v", err)
	}
	if forked.Thread.ID != "thread-fork" || forked.Thread.ForkedFromID == nil || forked.Thread.ForkedFromID.Value == nil || *forked.Thread.ForkedFromID.Value != "thread-1" {
		t.Fatalf("thread/fork response = %#v", forked)
	}
	if _, err := threads.ApproveGuardianDeniedAction(context.Background(), protocolv2.ThreadApproveGuardianDeniedActionParams{
		Event: protocolv2.JSONObject(map[string]protocolv2.JSONValue{
			"outcome": protocolv2.JSONString("deny"),
		}),
		ThreadID: "thread-1",
	}); err != nil {
		t.Fatalf("Threads().ApproveGuardianDeniedAction returned error: %v", err)
	}
	if _, err := threads.Archive(context.Background(), protocolv2.ThreadArchiveParams{ThreadID: "thread-1"}); err != nil {
		t.Fatalf("Threads().Archive returned error: %v", err)
	}
	if _, err := threads.CompactStart(context.Background(), protocolv2.ThreadCompactStartParams{ThreadID: "thread-1"}); err != nil {
		t.Fatalf("Threads().CompactStart returned error: %v", err)
	}
	if _, err := threads.InjectItems(context.Background(), protocolv2.ThreadInjectItemsParams{
		Items:    []protocolv2.JSONValue{protocolv2.JSONString("item-1")},
		ThreadID: "thread-1",
	}); err != nil {
		t.Fatalf("Threads().InjectItems returned error: %v", err)
	}
	list, err := threads.List(context.Background(), protocolv2.ThreadListParams{
		Archived:       protocolv2.Null[bool](),
		Cursor:         protocolv2.Value("cursor-1"),
		CWD:            protocolv2.Value(protocolv2.NewThreadListCwdFilterString("/workspace/facade")),
		Limit:          protocolv2.Value(uint32(10)),
		ModelProviders: protocolv2.Value([]string{"openai"}),
		SearchTerm:     protocolv2.Value("review"),
		SortDirection:  protocolv2.Value(protocolv2.SortDirectionDesc),
		SortKey:        protocolv2.Value(protocolv2.ThreadSortKeyUpdatedAt),
		SourceKinds:    protocolv2.Value([]protocolv2.ThreadSourceKind{protocolv2.ThreadSourceKindAppServer}),
		UseStateDbOnly: Bool(true),
	})
	if err != nil {
		t.Fatalf("Threads().List returned error: %v", err)
	}
	if len(list.Data) != 1 || list.Data[0].ID != "thread-list-1" || list.NextCursor == nil || list.NextCursor.Value == nil || *list.NextCursor.Value != "next-thread" {
		t.Fatalf("thread/list response = %#v", list)
	}
	if _, err := threads.List(context.Background(), protocolv2.ThreadListParams{
		CWD: protocolv2.Value(protocolv2.NewThreadListCwdFilterArray([]string{"/workspace/facade", "/workspace/other"})),
	}); err != nil {
		t.Fatalf("Threads().List with cwd array returned error: %v", err)
	}
	loaded, err := threads.LoadedList(context.Background(), protocolv2.ThreadLoadedListParams{
		Cursor: protocolv2.Null[string](),
		Limit:  protocolv2.Value(uint32(5)),
	})
	if err != nil {
		t.Fatalf("Threads().LoadedList returned error: %v", err)
	}
	if len(loaded.Data) != 1 || loaded.Data[0] != "thread-1" || loaded.NextCursor == nil || loaded.NextCursor.Value == nil || *loaded.NextCursor.Value != "loaded-next" {
		t.Fatalf("thread/loaded/list response = %#v", loaded)
	}
	metadata, err := threads.MetadataUpdate(context.Background(), protocolv2.ThreadMetadataUpdateParams{
		GitInfo: protocolv2.Value(protocolv2.ThreadMetadataGitInfoUpdateParams{
			Branch:    protocolv2.Value("main"),
			OriginURL: protocolv2.Null[string](),
			SHA:       protocolv2.Value("abc"),
		}),
		ThreadID: "thread-1",
	})
	if err != nil {
		t.Fatalf("Threads().MetadataUpdate returned error: %v", err)
	}
	if metadata.Thread.ID != "thread-1" {
		t.Fatalf("thread/metadata/update response = %#v", metadata)
	}
	if _, err := threads.NameSet(context.Background(), protocolv2.ThreadSetNameParams{Name: "Renamed", ThreadID: "thread-1"}); err != nil {
		t.Fatalf("Threads().NameSet returned error: %v", err)
	}
	read, err := threads.Read(context.Background(), protocolv2.ThreadReadParams{IncludeTurns: Bool(true), ThreadID: "thread-1"})
	if err != nil {
		t.Fatalf("Threads().Read returned error: %v", err)
	}
	if read.Thread.ID != "thread-1" {
		t.Fatalf("thread/read response = %#v", read)
	}
	rollback, err := threads.Rollback(context.Background(), protocolv2.ThreadRollbackParams{NumTurns: 1, ThreadID: "thread-1"})
	if err != nil {
		t.Fatalf("Threads().Rollback returned error: %v", err)
	}
	if rollback.Thread.ID != "thread-1" {
		t.Fatalf("thread/rollback response = %#v", rollback)
	}
	if _, err := threads.ShellCommand(context.Background(), protocolv2.ThreadShellCommandParams{Command: "echo ok", ThreadID: "thread-1"}); err != nil {
		t.Fatalf("Threads().ShellCommand returned error: %v", err)
	}
	unarchived, err := threads.Unarchive(context.Background(), protocolv2.ThreadUnarchiveParams{ThreadID: "thread-1"})
	if err != nil {
		t.Fatalf("Threads().Unarchive returned error: %v", err)
	}
	if unarchived.Thread.ID != "thread-1" {
		t.Fatalf("thread/unarchive response = %#v", unarchived)
	}
	unsubscribed, err := threads.Unsubscribe(context.Background(), protocolv2.ThreadUnsubscribeParams{ThreadID: "thread-1"})
	if err != nil {
		t.Fatalf("Threads().Unsubscribe returned error: %v", err)
	}
	if unsubscribed.Status != protocolv2.ThreadUnsubscribeStatusUnsubscribed {
		t.Fatalf("thread/unsubscribe response = %#v", unsubscribed)
	}
	schema := mustOutputSchema(t, `{"type":"object","required":["answer"],"properties":{"answer":{"type":"string"}}}`)
	turn, err := turns.Start(context.Background(), protocolv2.TurnStartParams{
		ThreadID:     "thread-1",
		Input:        []protocolv2.UserInput{protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "hello"})},
		OutputSchema: &schema,
	})
	if err != nil {
		t.Fatalf("Turns().Start returned error: %v", err)
	}
	if turn.Turn.ID != "turn-1" || turn.Turn.Status != protocolv2.TurnStatusInProgress {
		t.Fatalf("turn/start response = %#v", turn)
	}
	if _, err := turns.Interrupt(context.Background(), protocolv2.TurnInterruptParams{
		ThreadID: "thread-1",
		TurnID:   "turn-1",
	}); err != nil {
		t.Fatalf("Turns().Interrupt returned error: %v", err)
	}
	steered, err := turns.Steer(context.Background(), protocolv2.TurnSteerParams{
		ExpectedTurnID: "turn-1",
		Input:          []protocolv2.UserInput{protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "continue"})},
		ThreadID:       "thread-1",
	})
	if err != nil {
		t.Fatalf("Turns().Steer returned error: %v", err)
	}
	if steered.TurnID != "turn-1" {
		t.Fatalf("turn/steer response = %#v", steered)
	}

	records := readRecords(t, record)
	for _, method := range []string{
		protocolv2.MethodThreadStart,
		protocolv2.MethodThreadResume,
		protocolv2.MethodThreadFork,
		protocolv2.MethodThreadApproveGuardianDeniedAction,
		protocolv2.MethodThreadArchive,
		protocolv2.MethodThreadCompactStart,
		protocolv2.MethodThreadInjectItems,
		protocolv2.MethodThreadList,
		protocolv2.MethodThreadLoadedList,
		protocolv2.MethodThreadMetadataUpdate,
		protocolv2.MethodThreadNameSet,
		protocolv2.MethodThreadRead,
		protocolv2.MethodThreadRollback,
		protocolv2.MethodThreadShellCommand,
		protocolv2.MethodThreadUnarchive,
		protocolv2.MethodThreadUnsubscribe,
		protocolv2.MethodTurnInterrupt,
		protocolv2.MethodTurnStart,
		protocolv2.MethodTurnSteer,
	} {
		if firstRecord(records, "recv", method) == nil {
			t.Fatalf("missing facade method %s in records %#v", method, records)
		}
	}
	threadStart := firstRecord(records, "recv", protocolv2.MethodThreadStart)["params"].(map[string]any)
	if threadStart["model"] != "gpt-facade" || threadStart["cwd"] != "/workspace/facade" {
		t.Fatalf("thread/start params = %#v", threadStart)
	}
	approve := firstRecord(records, "recv", protocolv2.MethodThreadApproveGuardianDeniedAction)["params"].(map[string]any)
	approveEvent := approve["event"].(map[string]any)
	if approve["threadId"] != "thread-1" || approveEvent["outcome"] != "deny" {
		t.Fatalf("thread/approveGuardianDeniedAction params = %#v", approve)
	}
	listRecords := recordsByMethod(records, "recv", protocolv2.MethodThreadList)
	if len(listRecords) != 2 {
		t.Fatalf("thread/list records = %#v, want two calls", listRecords)
	}
	threadList := listRecords[0]["params"].(map[string]any)
	if threadList["archived"] != nil || threadList["cursor"] != "cursor-1" || threadList["cwd"] != "/workspace/facade" ||
		threadList["limit"] != float64(10) || threadList["sortDirection"] != "desc" || threadList["sortKey"] != "updated_at" ||
		threadList["useStateDbOnly"] != true {
		t.Fatalf("thread/list params = %#v", threadList)
	}
	threadListArray := listRecords[1]["params"].(map[string]any)
	threadListArrayCWD := threadListArray["cwd"].([]any)
	if len(threadListArrayCWD) != 2 || threadListArrayCWD[0] != "/workspace/facade" || threadListArrayCWD[1] != "/workspace/other" {
		t.Fatalf("thread/list cwd array params = %#v", threadListArray)
	}
	metadataParams := firstRecord(records, "recv", protocolv2.MethodThreadMetadataUpdate)["params"].(map[string]any)
	metadataGitInfo := metadataParams["gitInfo"].(map[string]any)
	if metadataParams["threadId"] != "thread-1" || metadataGitInfo["branch"] != "main" ||
		metadataGitInfo["originUrl"] != nil || metadataGitInfo["sha"] != "abc" {
		t.Fatalf("thread/metadata/update params = %#v", metadataParams)
	}
	turnStart := firstRecord(records, "recv", protocolv2.MethodTurnStart)["params"].(map[string]any)
	if turnStart["threadId"] != "thread-1" || turnStart["outputSchema"] == nil {
		t.Fatalf("turn/start params = %#v", turnStart)
	}
	turnInterrupt := firstRecord(records, "recv", protocolv2.MethodTurnInterrupt)["params"].(map[string]any)
	if turnInterrupt["threadId"] != "thread-1" || turnInterrupt["turnId"] != "turn-1" {
		t.Fatalf("turn/interrupt params = %#v", turnInterrupt)
	}
	turnSteer := firstRecord(records, "recv", protocolv2.MethodTurnSteer)["params"].(map[string]any)
	turnSteerInput := turnSteer["input"].([]any)
	turnSteerText := turnSteerInput[0].(map[string]any)
	if turnSteer["threadId"] != "thread-1" || turnSteer["expectedTurnId"] != "turn-1" ||
		turnSteerText["type"] != "text" || turnSteerText["text"] != "continue" ||
		turnSteer["responsesapiClientMetadata"] != nil {
		t.Fatalf("turn/steer params = %#v", turnSteer)
	}
}

func TestThreadGoalProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	threads := root.Threads()
	set, err := threads.GoalSet(context.Background(), protocolv2.ThreadGoalSetParams{
		Objective:   protocolv2.Value("ship protocol coverage"),
		Status:      protocolv2.Value(protocolv2.ThreadGoalStatusActive),
		ThreadID:    "thread-1",
		TokenBudget: protocolv2.Value(int64(4096)),
	})
	if err != nil {
		t.Fatalf("Threads().GoalSet returned error after experimental opt-in: %v", err)
	}
	if set.Goal.ThreadID != "thread-1" || set.Goal.Objective != "ship protocol coverage" ||
		set.Goal.Status != protocolv2.ThreadGoalStatusActive ||
		set.Goal.TokenBudget == nil || set.Goal.TokenBudget.Value == nil || *set.Goal.TokenBudget.Value != 4096 {
		t.Fatalf("thread/goal/set response = %#v", set)
	}
	get, err := threads.GoalGet(context.Background(), protocolv2.ThreadGoalGetParams{ThreadID: "thread-1"})
	if err != nil {
		t.Fatalf("Threads().GoalGet returned error after experimental opt-in: %v", err)
	}
	if get.Goal == nil || get.Goal.Value == nil || get.Goal.Value.ThreadID != "thread-1" ||
		get.Goal.Value.Objective != "ship protocol coverage" ||
		get.Goal.Value.Status != protocolv2.ThreadGoalStatusActive {
		t.Fatalf("thread/goal/get response = %#v", get)
	}
	clear, err := threads.GoalClear(context.Background(), protocolv2.ThreadGoalClearParams{ThreadID: "thread-1"})
	if err != nil {
		t.Fatalf("Threads().GoalClear returned error after experimental opt-in: %v", err)
	}
	if !clear.Cleared {
		t.Fatalf("thread/goal/clear response = %#v", clear)
	}

	records := readRecords(t, record)
	for _, method := range []string{
		protocolv2.MethodThreadGoalSet,
		protocolv2.MethodThreadGoalGet,
		protocolv2.MethodThreadGoalClear,
	} {
		if firstRecord(records, "recv", method) == nil {
			t.Fatalf("missing thread goal method %s in records %#v", method, records)
		}
	}
	setParams := firstRecord(records, "recv", protocolv2.MethodThreadGoalSet)["params"].(map[string]any)
	if setParams["threadId"] != "thread-1" || setParams["objective"] != "ship protocol coverage" ||
		setParams["status"] != "active" || setParams["tokenBudget"] != float64(4096) {
		t.Fatalf("thread/goal/set params = %#v", setParams)
	}
	getParams := firstRecord(records, "recv", protocolv2.MethodThreadGoalGet)["params"].(map[string]any)
	if getParams["threadId"] != "thread-1" {
		t.Fatalf("thread/goal/get params = %#v", getParams)
	}
	clearParams := firstRecord(records, "recv", protocolv2.MethodThreadGoalClear)["params"].(map[string]any)
	if clearParams["threadId"] != "thread-1" {
		t.Fatalf("thread/goal/clear params = %#v", clearParams)
	}
}

func TestThreadGoalProtocolFamilyFacadeRejectsMalformedTypedResponses(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		call    func(Threads) error
		wantSub string
	}{
		{
			name:   "set",
			method: protocolv2.MethodThreadGoalSet,
			call: func(threads Threads) error {
				_, err := threads.GoalSet(context.Background(), protocolv2.ThreadGoalSetParams{ThreadID: "thread-1"})
				return err
			},
			wantSub: "ThreadGoal.createdAt",
		},
		{
			name:   "get",
			method: protocolv2.MethodThreadGoalGet,
			call: func(threads Threads) error {
				_, err := threads.GoalGet(context.Background(), protocolv2.ThreadGoalGetParams{ThreadID: "thread-1"})
				return err
			},
			wantSub: "ThreadGoal.createdAt",
		},
		{
			name:   "clear",
			method: protocolv2.MethodThreadGoalClear,
			call: func(threads Threads) error {
				_, err := threads.GoalClear(context.Background(), protocolv2.ThreadGoalClearParams{ThreadID: "thread-1"})
				return err
			},
			wantSub: "ThreadGoalClearResponse.cleared",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := tempRecord(t)
			t.Setenv("CODEXSDK_FAKE_RECORD", record)
			root, err := New(ClientOptions{
				CWD:          t.TempDir(),
				Command:      fakeCommand("thread-goal-malformed-response"),
				Capabilities: ClientCapabilities{ExperimentalAPI: true},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()

			err = tc.call(root.Threads())
			if err == nil {
				t.Fatalf("%s accepted malformed response", tc.method)
			}
			if !strings.Contains(err.Error(), "decode "+tc.method+" response") ||
				!strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("malformed %s response error = %v", tc.method, err)
			}
			if firstRecord(readRecords(t, record), "recv", tc.method) == nil {
				t.Fatalf("%s was not sent before malformed response decode", tc.method)
			}
		})
	}
}

func TestThreadSupportProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	threads := root.Threads()
	incremented, err := threads.IncrementElicitation(context.Background(), protocolv2.ThreadIncrementElicitationParams{ThreadID: "thread-1"})
	if err != nil {
		t.Fatalf("Threads().IncrementElicitation returned error after experimental opt-in: %v", err)
	}
	if incremented.Count != 3 || !incremented.Paused {
		t.Fatalf("thread/increment_elicitation response = %#v", incremented)
	}
	decremented, err := threads.DecrementElicitation(context.Background(), protocolv2.ThreadDecrementElicitationParams{ThreadID: "thread-1"})
	if err != nil {
		t.Fatalf("Threads().DecrementElicitation returned error after experimental opt-in: %v", err)
	}
	if decremented.Count != 2 || decremented.Paused {
		t.Fatalf("thread/decrement_elicitation response = %#v", decremented)
	}
	if _, err := threads.MemoryModeSet(context.Background(), protocolv2.ThreadMemoryModeSetParams{
		Mode:     protocolv2.ThreadMemoryModeEnabled,
		ThreadID: "thread-1",
	}); err != nil {
		t.Fatalf("Threads().MemoryModeSet returned error after experimental opt-in: %v", err)
	}
	if _, err := threads.BackgroundTerminalsClean(context.Background(), protocolv2.ThreadBackgroundTerminalsCleanParams{ThreadID: "thread-1"}); err != nil {
		t.Fatalf("Threads().BackgroundTerminalsClean returned error after experimental opt-in: %v", err)
	}

	records := readRecords(t, record)
	for _, method := range []string{
		protocolv2.MethodThreadIncrementElicitation,
		protocolv2.MethodThreadDecrementElicitation,
		protocolv2.MethodThreadMemoryModeSet,
		protocolv2.MethodThreadBackgroundTerminalsClean,
	} {
		if firstRecord(records, "recv", method) == nil {
			t.Fatalf("missing thread support method %s in records %#v", method, records)
		}
	}
	incrementParams := firstRecord(records, "recv", protocolv2.MethodThreadIncrementElicitation)["params"].(map[string]any)
	if incrementParams["threadId"] != "thread-1" {
		t.Fatalf("thread/increment_elicitation params = %#v", incrementParams)
	}
	decrementParams := firstRecord(records, "recv", protocolv2.MethodThreadDecrementElicitation)["params"].(map[string]any)
	if decrementParams["threadId"] != "thread-1" {
		t.Fatalf("thread/decrement_elicitation params = %#v", decrementParams)
	}
	memoryModeParams := firstRecord(records, "recv", protocolv2.MethodThreadMemoryModeSet)["params"].(map[string]any)
	if memoryModeParams["threadId"] != "thread-1" || memoryModeParams["mode"] != "enabled" {
		t.Fatalf("thread/memoryMode/set params = %#v", memoryModeParams)
	}
	cleanParams := firstRecord(records, "recv", protocolv2.MethodThreadBackgroundTerminalsClean)["params"].(map[string]any)
	if cleanParams["threadId"] != "thread-1" {
		t.Fatalf("thread/backgroundTerminals/clean params = %#v", cleanParams)
	}
}

func TestThreadSupportProtocolFamilyFacadeRejectsExperimentalMethodsBeforeWriteUnlessOptedIn(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	cases := []struct {
		name   string
		method string
		call   func() error
	}{
		{
			name:   "increment",
			method: protocolv2.MethodThreadIncrementElicitation,
			call: func() error {
				_, err := root.Threads().IncrementElicitation(context.Background(), protocolv2.ThreadIncrementElicitationParams{ThreadID: "thread-1"})
				return err
			},
		},
		{
			name:   "decrement",
			method: protocolv2.MethodThreadDecrementElicitation,
			call: func() error {
				_, err := root.Threads().DecrementElicitation(context.Background(), protocolv2.ThreadDecrementElicitationParams{ThreadID: "thread-1"})
				return err
			},
		},
		{
			name:   "memoryModeSet",
			method: protocolv2.MethodThreadMemoryModeSet,
			call: func() error {
				_, err := root.Threads().MemoryModeSet(context.Background(), protocolv2.ThreadMemoryModeSetParams{
					Mode:     protocolv2.ThreadMemoryModeEnabled,
					ThreadID: "thread-1",
				})
				return err
			},
		},
		{
			name:   "backgroundTerminalsClean",
			method: protocolv2.MethodThreadBackgroundTerminalsClean,
			call: func() error {
				_, err := root.Threads().BackgroundTerminalsClean(context.Background(), protocolv2.ThreadBackgroundTerminalsCleanParams{ThreadID: "thread-1"})
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatalf("%s accepted experimental method without opt-in", tc.method)
			}
			if !strings.Contains(err.Error(), "experimental app-server method \""+tc.method+"\" requires ClientCapabilities.ExperimentalAPI") {
				t.Fatalf("experimental %s error = %v", tc.method, err)
			}
		})
	}
	records := readRecords(t, record)
	for _, method := range []string{
		protocolv2.MethodThreadIncrementElicitation,
		protocolv2.MethodThreadDecrementElicitation,
		protocolv2.MethodThreadMemoryModeSet,
		protocolv2.MethodThreadBackgroundTerminalsClean,
	} {
		if firstRecord(records, "recv", method) != nil {
			t.Fatalf("%s was sent after experimental method guard failure", method)
		}
	}
}

func TestThreadSupportProtocolFamilyFacadePreflightFailureDoesNotWriteRequest(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Threads().MemoryModeSet(context.Background(), protocolv2.ThreadMemoryModeSetParams{
		Mode:     protocolv2.ThreadMemoryMode("bogus"),
		ThreadID: "thread-1",
	})
	if err == nil {
		t.Fatal("Threads().MemoryModeSet succeeded with invalid memory mode")
	}
	if !strings.Contains(err.Error(), "encode thread/memoryMode/set params") ||
		!strings.Contains(err.Error(), `invalid ThreadMemoryMode enum value "bogus"`) {
		t.Fatalf("thread/memoryMode/set preflight error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadMemoryModeSet) != nil {
		t.Fatal("thread/memoryMode/set was sent after preflight failure")
	}
}

func TestThreadSupportProtocolFamilyFacadeRejectsMalformedTypedResponses(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		call    func(Threads) error
		wantSub string
	}{
		{
			name:   "increment",
			method: protocolv2.MethodThreadIncrementElicitation,
			call: func(threads Threads) error {
				_, err := threads.IncrementElicitation(context.Background(), protocolv2.ThreadIncrementElicitationParams{ThreadID: "thread-1"})
				return err
			},
			wantSub: "ThreadIncrementElicitationResponse.count",
		},
		{
			name:   "decrement",
			method: protocolv2.MethodThreadDecrementElicitation,
			call: func(threads Threads) error {
				_, err := threads.DecrementElicitation(context.Background(), protocolv2.ThreadDecrementElicitationParams{ThreadID: "thread-1"})
				return err
			},
			wantSub: "ThreadDecrementElicitationResponse.count",
		},
		{
			name:   "memoryModeSet",
			method: protocolv2.MethodThreadMemoryModeSet,
			call: func(threads Threads) error {
				_, err := threads.MemoryModeSet(context.Background(), protocolv2.ThreadMemoryModeSetParams{
					Mode:     protocolv2.ThreadMemoryModeEnabled,
					ThreadID: "thread-1",
				})
				return err
			},
			wantSub: `ThreadMemoryModeSetResponse: unknown field "extra"`,
		},
		{
			name:   "backgroundTerminalsClean",
			method: protocolv2.MethodThreadBackgroundTerminalsClean,
			call: func(threads Threads) error {
				_, err := threads.BackgroundTerminalsClean(context.Background(), protocolv2.ThreadBackgroundTerminalsCleanParams{ThreadID: "thread-1"})
				return err
			},
			wantSub: `ThreadBackgroundTerminalsCleanResponse: unknown field "extra"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := tempRecord(t)
			t.Setenv("CODEXSDK_FAKE_RECORD", record)
			root, err := New(ClientOptions{
				CWD:          t.TempDir(),
				Command:      fakeCommand("thread-support-malformed-response"),
				Capabilities: ClientCapabilities{ExperimentalAPI: true},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()

			err = tc.call(root.Threads())
			if err == nil {
				t.Fatalf("%s accepted malformed response", tc.method)
			}
			if !strings.Contains(err.Error(), "decode "+tc.method+" response") ||
				!strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("malformed %s response error = %v", tc.method, err)
			}
			if firstRecord(readRecords(t, record), "recv", tc.method) == nil {
				t.Fatalf("%s was not sent before malformed response decode", tc.method)
			}
		})
	}
}

func TestThreadTurnsProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	threads := root.Threads()
	turnsList, err := threads.TurnsList(context.Background(), protocolv2.ThreadTurnsListParams{
		Cursor:        protocolv2.Value("cursor-1"),
		ItemsView:     protocolv2.Value(protocolv2.TurnItemsViewSummary),
		Limit:         protocolv2.Value(uint32(5)),
		SortDirection: protocolv2.Value(protocolv2.SortDirectionDesc),
		ThreadID:      "thread-1",
	})
	if err != nil {
		t.Fatalf("Threads().TurnsList returned error after experimental opt-in: %v", err)
	}
	if len(turnsList.Data) != 1 || turnsList.Data[0].ID != "turn-list-1" ||
		turnsList.NextCursor == nil || turnsList.NextCursor.Value == nil || *turnsList.NextCursor.Value != "next-turn" {
		t.Fatalf("thread/turns/list response = %#v", turnsList)
	}
	itemsList, err := threads.TurnsItemsList(context.Background(), protocolv2.ThreadTurnsItemsListParams{
		Cursor:        protocolv2.Null[string](),
		Limit:         protocolv2.Value(uint32(10)),
		SortDirection: protocolv2.Value(protocolv2.SortDirectionAsc),
		ThreadID:      "thread-1",
		TurnID:        "turn-list-1",
	})
	if err != nil {
		t.Fatalf("Threads().TurnsItemsList returned error after experimental opt-in: %v", err)
	}
	if len(itemsList.Data) != 1 || itemsList.Data[0].Kind() != protocolv2.ThreadItemKindAgentMessage ||
		itemsList.NextCursor == nil || itemsList.NextCursor.Value == nil || *itemsList.NextCursor.Value != "next-item" {
		t.Fatalf("thread/turns/items/list response = %#v", itemsList)
	}

	records := readRecords(t, record)
	turnsParams := firstRecord(records, "recv", protocolv2.MethodThreadTurnsList)["params"].(map[string]any)
	if turnsParams["threadId"] != "thread-1" || turnsParams["cursor"] != "cursor-1" ||
		turnsParams["itemsView"] != "summary" || turnsParams["limit"] != float64(5) ||
		turnsParams["sortDirection"] != "desc" {
		t.Fatalf("thread/turns/list params = %#v", turnsParams)
	}
	itemsParams := firstRecord(records, "recv", protocolv2.MethodThreadTurnsItemsList)["params"].(map[string]any)
	if itemsParams["threadId"] != "thread-1" || itemsParams["turnId"] != "turn-list-1" ||
		itemsParams["cursor"] != nil || itemsParams["limit"] != float64(10) ||
		itemsParams["sortDirection"] != "asc" {
		t.Fatalf("thread/turns/items/list params = %#v", itemsParams)
	}
}

func TestThreadTurnsProtocolFamilyFacadeRejectsExperimentalMethodsBeforeWriteUnlessOptedIn(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	cases := []struct {
		name   string
		method string
		call   func() error
	}{
		{
			name:   "turnsList",
			method: protocolv2.MethodThreadTurnsList,
			call: func() error {
				_, err := root.Threads().TurnsList(context.Background(), protocolv2.ThreadTurnsListParams{ThreadID: "thread-1"})
				return err
			},
		},
		{
			name:   "turnsItemsList",
			method: protocolv2.MethodThreadTurnsItemsList,
			call: func() error {
				_, err := root.Threads().TurnsItemsList(context.Background(), protocolv2.ThreadTurnsItemsListParams{
					ThreadID: "thread-1",
					TurnID:   "turn-list-1",
				})
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatalf("%s accepted experimental method without opt-in", tc.method)
			}
			if !strings.Contains(err.Error(), "experimental app-server method \""+tc.method+"\" requires ClientCapabilities.ExperimentalAPI") {
				t.Fatalf("experimental %s error = %v", tc.method, err)
			}
		})
	}
	records := readRecords(t, record)
	for _, method := range []string{
		protocolv2.MethodThreadTurnsList,
		protocolv2.MethodThreadTurnsItemsList,
	} {
		if firstRecord(records, "recv", method) != nil {
			t.Fatalf("%s was sent after experimental method guard failure", method)
		}
	}
}

func TestThreadTurnsProtocolFamilyFacadePreflightFailureDoesNotWriteRequest(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Threads().TurnsList(context.Background(), protocolv2.ThreadTurnsListParams{
		SortDirection: protocolv2.Value(protocolv2.SortDirection("sideways")),
		ThreadID:      "thread-1",
	})
	if err == nil {
		t.Fatal("Threads().TurnsList succeeded with invalid sortDirection")
	}
	if !strings.Contains(err.Error(), "encode thread/turns/list params") ||
		!strings.Contains(err.Error(), `invalid SortDirection enum value "sideways"`) {
		t.Fatalf("thread/turns/list preflight error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadTurnsList) != nil {
		t.Fatal("thread/turns/list was sent after preflight failure")
	}
}

func TestThreadTurnsProtocolFamilyFacadeRejectsMalformedTypedResponses(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		call    func(Threads) error
		wantSub string
	}{
		{
			name:   "turnsList",
			method: protocolv2.MethodThreadTurnsList,
			call: func(threads Threads) error {
				_, err := threads.TurnsList(context.Background(), protocolv2.ThreadTurnsListParams{ThreadID: "thread-1"})
				return err
			},
			wantSub: "ThreadTurnsListResponse.data",
		},
		{
			name:   "turnsItemsList",
			method: protocolv2.MethodThreadTurnsItemsList,
			call: func(threads Threads) error {
				_, err := threads.TurnsItemsList(context.Background(), protocolv2.ThreadTurnsItemsListParams{
					ThreadID: "thread-1",
					TurnID:   "turn-list-1",
				})
				return err
			},
			wantSub: "ThreadTurnsItemsListResponse.data",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := tempRecord(t)
			t.Setenv("CODEXSDK_FAKE_RECORD", record)
			root, err := New(ClientOptions{
				CWD:          t.TempDir(),
				Command:      fakeCommand("thread-turns-malformed-response"),
				Capabilities: ClientCapabilities{ExperimentalAPI: true},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()

			err = tc.call(root.Threads())
			if err == nil {
				t.Fatalf("%s accepted malformed response", tc.method)
			}
			if !strings.Contains(err.Error(), "decode "+tc.method+" response") ||
				!strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("malformed %s response error = %v", tc.method, err)
			}
			if firstRecord(readRecords(t, record), "recv", tc.method) == nil {
				t.Fatalf("%s was not sent before malformed response decode", tc.method)
			}
		})
	}
}

func TestThreadRealtimeProtocolFamilyFacadeSendsTypedMethodsAndDecodesResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	threads := root.Threads()
	if _, err := threads.RealtimeStart(context.Background(), protocolv2.ThreadRealtimeStartParams{
		OutputModality:    protocolv2.RealtimeOutputModalityAudio,
		Prompt:            protocolv2.Null[string](),
		RealtimeSessionID: protocolv2.Value("session-1"),
		ThreadID:          "thread-1",
		Transport:         protocolv2.Value(protocolv2.NewThreadRealtimeStartTransportWebrtc(protocolv2.ThreadRealtimeStartTransportWebrtc{SDP: "offer"})),
		Voice:             protocolv2.Value(protocolv2.RealtimeVoiceMarin),
	}); err != nil {
		t.Fatalf("Threads().RealtimeStart returned error after experimental opt-in: %v", err)
	}
	if _, err := threads.RealtimeAppendAudio(context.Background(), protocolv2.ThreadRealtimeAppendAudioParams{
		Audio:    facadeRealtimeAudioChunk(),
		ThreadID: "thread-1",
	}); err != nil {
		t.Fatalf("Threads().RealtimeAppendAudio returned error after experimental opt-in: %v", err)
	}
	if _, err := threads.RealtimeAppendText(context.Background(), protocolv2.ThreadRealtimeAppendTextParams{
		Text:     "hello",
		ThreadID: "thread-1",
	}); err != nil {
		t.Fatalf("Threads().RealtimeAppendText returned error after experimental opt-in: %v", err)
	}
	voices, err := threads.RealtimeListVoices(context.Background(), protocolv2.ThreadRealtimeListVoicesParams{})
	if err != nil {
		t.Fatalf("Threads().RealtimeListVoices returned error after experimental opt-in: %v", err)
	}
	if voices.Voices.DefaultV1 != protocolv2.RealtimeVoiceAlloy || voices.Voices.DefaultV2 != protocolv2.RealtimeVoiceMarin ||
		len(voices.Voices.V1) != 1 || voices.Voices.V1[0] != protocolv2.RealtimeVoiceAlloy ||
		len(voices.Voices.V2) != 1 || voices.Voices.V2[0] != protocolv2.RealtimeVoiceMarin {
		t.Fatalf("thread/realtime/listVoices response = %#v", voices)
	}
	if _, err := threads.RealtimeStop(context.Background(), protocolv2.ThreadRealtimeStopParams{ThreadID: "thread-1"}); err != nil {
		t.Fatalf("Threads().RealtimeStop returned error after experimental opt-in: %v", err)
	}

	records := readRecords(t, record)
	for _, method := range []string{
		protocolv2.MethodThreadRealtimeStart,
		protocolv2.MethodThreadRealtimeAppendAudio,
		protocolv2.MethodThreadRealtimeAppendText,
		protocolv2.MethodThreadRealtimeListVoices,
		protocolv2.MethodThreadRealtimeStop,
	} {
		if firstRecord(records, "recv", method) == nil {
			t.Fatalf("missing realtime method %s in records %#v", method, records)
		}
	}
	startParams := firstRecord(records, "recv", protocolv2.MethodThreadRealtimeStart)["params"].(map[string]any)
	startTransport := startParams["transport"].(map[string]any)
	if startParams["threadId"] != "thread-1" || startParams["outputModality"] != "audio" ||
		startParams["prompt"] != nil || startParams["realtimeSessionId"] != "session-1" ||
		startParams["voice"] != "marin" || startTransport["type"] != "webrtc" || startTransport["sdp"] != "offer" {
		t.Fatalf("thread/realtime/start params = %#v", startParams)
	}
	audioParams := firstRecord(records, "recv", protocolv2.MethodThreadRealtimeAppendAudio)["params"].(map[string]any)
	audio := audioParams["audio"].(map[string]any)
	if audioParams["threadId"] != "thread-1" || audio["data"] != "base64" || audio["itemId"] != "item-1" ||
		audio["numChannels"] != float64(2) || audio["sampleRate"] != float64(24000) ||
		audio["samplesPerChannel"] != float64(480) {
		t.Fatalf("thread/realtime/appendAudio params = %#v", audioParams)
	}
	textParams := firstRecord(records, "recv", protocolv2.MethodThreadRealtimeAppendText)["params"].(map[string]any)
	if textParams["threadId"] != "thread-1" || textParams["text"] != "hello" {
		t.Fatalf("thread/realtime/appendText params = %#v", textParams)
	}
	voicesParams := firstRecord(records, "recv", protocolv2.MethodThreadRealtimeListVoices)["params"].(map[string]any)
	if len(voicesParams) != 0 {
		t.Fatalf("thread/realtime/listVoices params = %#v, want empty object", voicesParams)
	}
	stopParams := firstRecord(records, "recv", protocolv2.MethodThreadRealtimeStop)["params"].(map[string]any)
	if stopParams["threadId"] != "thread-1" {
		t.Fatalf("thread/realtime/stop params = %#v", stopParams)
	}
}

func TestThreadRealtimeProtocolFamilyFacadeRejectsExperimentalMethodsBeforeWriteUnlessOptedIn(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	cases := []struct {
		name   string
		method string
		call   func() error
	}{
		{
			name:   "start",
			method: protocolv2.MethodThreadRealtimeStart,
			call: func() error {
				_, err := root.Threads().RealtimeStart(context.Background(), protocolv2.ThreadRealtimeStartParams{
					OutputModality: protocolv2.RealtimeOutputModalityAudio,
					ThreadID:       "thread-1",
				})
				return err
			},
		},
		{
			name:   "appendAudio",
			method: protocolv2.MethodThreadRealtimeAppendAudio,
			call: func() error {
				_, err := root.Threads().RealtimeAppendAudio(context.Background(), protocolv2.ThreadRealtimeAppendAudioParams{
					Audio:    facadeRealtimeAudioChunk(),
					ThreadID: "thread-1",
				})
				return err
			},
		},
		{
			name:   "appendText",
			method: protocolv2.MethodThreadRealtimeAppendText,
			call: func() error {
				_, err := root.Threads().RealtimeAppendText(context.Background(), protocolv2.ThreadRealtimeAppendTextParams{
					Text:     "hello",
					ThreadID: "thread-1",
				})
				return err
			},
		},
		{
			name:   "listVoices",
			method: protocolv2.MethodThreadRealtimeListVoices,
			call: func() error {
				_, err := root.Threads().RealtimeListVoices(context.Background(), protocolv2.ThreadRealtimeListVoicesParams{})
				return err
			},
		},
		{
			name:   "stop",
			method: protocolv2.MethodThreadRealtimeStop,
			call: func() error {
				_, err := root.Threads().RealtimeStop(context.Background(), protocolv2.ThreadRealtimeStopParams{ThreadID: "thread-1"})
				return err
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatalf("%s accepted experimental method without opt-in", tc.method)
			}
			if !strings.Contains(err.Error(), "experimental app-server method \""+tc.method+"\" requires ClientCapabilities.ExperimentalAPI") {
				t.Fatalf("experimental %s error = %v", tc.method, err)
			}
		})
	}
	records := readRecords(t, record)
	for _, method := range []string{
		protocolv2.MethodThreadRealtimeStart,
		protocolv2.MethodThreadRealtimeAppendAudio,
		protocolv2.MethodThreadRealtimeAppendText,
		protocolv2.MethodThreadRealtimeListVoices,
		protocolv2.MethodThreadRealtimeStop,
	} {
		if firstRecord(records, "recv", method) != nil {
			t.Fatalf("%s was sent after experimental method guard failure", method)
		}
	}
}

func TestThreadRealtimeProtocolFamilyFacadePreflightFailureDoesNotWriteRequest(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Threads().RealtimeStart(context.Background(), protocolv2.ThreadRealtimeStartParams{
		OutputModality: protocolv2.RealtimeOutputModality("video"),
		ThreadID:       "thread-1",
	})
	if err == nil {
		t.Fatal("Threads().RealtimeStart succeeded with invalid outputModality")
	}
	if !strings.Contains(err.Error(), "encode thread/realtime/start params") ||
		!strings.Contains(err.Error(), `invalid RealtimeOutputModality enum value "video"`) {
		t.Fatalf("thread/realtime/start preflight error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadRealtimeStart) != nil {
		t.Fatal("thread/realtime/start was sent after preflight failure")
	}
}

func TestThreadRealtimeProtocolFamilyFacadeRejectsMalformedTypedResponses(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		call    func(Threads) error
		wantSub string
	}{
		{
			name:   "start",
			method: protocolv2.MethodThreadRealtimeStart,
			call: func(threads Threads) error {
				_, err := threads.RealtimeStart(context.Background(), protocolv2.ThreadRealtimeStartParams{
					OutputModality: protocolv2.RealtimeOutputModalityAudio,
					ThreadID:       "thread-1",
				})
				return err
			},
			wantSub: `ThreadRealtimeStartResponse: unknown field "extra"`,
		},
		{
			name:   "appendAudio",
			method: protocolv2.MethodThreadRealtimeAppendAudio,
			call: func(threads Threads) error {
				_, err := threads.RealtimeAppendAudio(context.Background(), protocolv2.ThreadRealtimeAppendAudioParams{
					Audio:    facadeRealtimeAudioChunk(),
					ThreadID: "thread-1",
				})
				return err
			},
			wantSub: `ThreadRealtimeAppendAudioResponse: unknown field "extra"`,
		},
		{
			name:   "appendText",
			method: protocolv2.MethodThreadRealtimeAppendText,
			call: func(threads Threads) error {
				_, err := threads.RealtimeAppendText(context.Background(), protocolv2.ThreadRealtimeAppendTextParams{
					Text:     "hello",
					ThreadID: "thread-1",
				})
				return err
			},
			wantSub: `ThreadRealtimeAppendTextResponse: unknown field "extra"`,
		},
		{
			name:   "listVoices",
			method: protocolv2.MethodThreadRealtimeListVoices,
			call: func(threads Threads) error {
				_, err := threads.RealtimeListVoices(context.Background(), protocolv2.ThreadRealtimeListVoicesParams{})
				return err
			},
			wantSub: "ThreadRealtimeListVoicesResponse.voices",
		},
		{
			name:   "stop",
			method: protocolv2.MethodThreadRealtimeStop,
			call: func(threads Threads) error {
				_, err := threads.RealtimeStop(context.Background(), protocolv2.ThreadRealtimeStopParams{ThreadID: "thread-1"})
				return err
			},
			wantSub: `ThreadRealtimeStopResponse: unknown field "extra"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := tempRecord(t)
			t.Setenv("CODEXSDK_FAKE_RECORD", record)
			root, err := New(ClientOptions{
				CWD:          t.TempDir(),
				Command:      fakeCommand("thread-realtime-malformed-response"),
				Capabilities: ClientCapabilities{ExperimentalAPI: true},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()

			err = tc.call(root.Threads())
			if err == nil {
				t.Fatalf("%s accepted malformed response", tc.method)
			}
			if !strings.Contains(err.Error(), "decode "+tc.method+" response") ||
				!strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("malformed %s response error = %v", tc.method, err)
			}
			if firstRecord(readRecords(t, record), "recv", tc.method) == nil {
				t.Fatalf("%s was not sent before malformed response decode", tc.method)
			}
		})
	}
}

func TestProtocolFacadePreflightFailureDoesNotWriteRequest(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Turns().Start(context.Background(), protocolv2.TurnStartParams{
		ThreadID: "thread-1",
	})
	if err == nil {
		t.Fatal("Turns().Start succeeded with nil required input")
	}
	if !strings.Contains(err.Error(), "encode turn/start params") ||
		!strings.Contains(err.Error(), "TurnStartParams.input: nil is not allowed") {
		t.Fatalf("preflight error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodTurnStart) != nil {
		t.Fatal("turn/start was sent after preflight failure")
	}
	_, err = root.Turns().Steer(context.Background(), protocolv2.TurnSteerParams{
		ExpectedTurnID: "turn-1",
		ThreadID:       "thread-1",
	})
	if err == nil {
		t.Fatal("Turns().Steer succeeded with nil required input")
	}
	if !strings.Contains(err.Error(), "encode turn/steer params") ||
		!strings.Contains(err.Error(), "TurnSteerParams.input: nil is not allowed") {
		t.Fatalf("turn/steer preflight error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodTurnSteer) != nil {
		t.Fatal("turn/steer was sent after preflight failure")
	}
	_, err = root.WindowsSandbox().SetupStart(context.Background(), protocolv2.WindowsSandboxSetupStartParams{})
	if err == nil {
		t.Fatal("WindowsSandbox().SetupStart succeeded with missing setup mode")
	}
	if !strings.Contains(err.Error(), "encode windowsSandbox/setupStart params") ||
		!strings.Contains(err.Error(), "WindowsSandboxSetupMode") {
		t.Fatalf("windows setup preflight error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodWindowsSandboxSetupStart) != nil {
		t.Fatal("windowsSandbox/setupStart was sent after preflight failure")
	}
	_, err = root.Config().BatchWrite(context.Background(), protocolv2.ConfigBatchWriteParams{})
	if err == nil {
		t.Fatal("Config().BatchWrite succeeded with nil required edits")
	}
	if !strings.Contains(err.Error(), "encode config/batchWrite params") ||
		!strings.Contains(err.Error(), "ConfigBatchWriteParams.edits: nil is not allowed") {
		t.Fatalf("config batchWrite preflight error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodConfigBatchWrite) != nil {
		t.Fatal("config/batchWrite was sent after preflight failure")
	}
	_, err = root.ExperimentalFeatures().EnablementSet(context.Background(), protocolv2.ExperimentalFeatureEnablementSetParams{})
	if err == nil {
		t.Fatal("ExperimentalFeatures().EnablementSet succeeded with nil required enablement")
	}
	if !strings.Contains(err.Error(), "encode experimentalFeature/enablement/set params") ||
		!strings.Contains(err.Error(), "ExperimentalFeatureEnablementSetParams.enablement: nil is not allowed") {
		t.Fatalf("experimental feature enablement preflight error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodExperimentalFeatureEnablementSet) != nil {
		t.Fatal("experimentalFeature/enablement/set was sent after preflight failure")
	}
	_, err = root.ExternalAgentConfigs().Import(context.Background(), protocolv2.ExternalAgentConfigImportParams{})
	if err == nil {
		t.Fatal("ExternalAgentConfigs().Import succeeded with nil required migrationItems")
	}
	if !strings.Contains(err.Error(), "encode externalAgentConfig/import params") ||
		!strings.Contains(err.Error(), "ExternalAgentConfigImportParams.migrationItems: nil is not allowed") {
		t.Fatalf("external agent config import preflight error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodExternalAgentConfigImport) != nil {
		t.Fatal("externalAgentConfig/import was sent after preflight failure")
	}
	_ = root.Close()

	record = tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err = New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	_, err = root.FuzzyFileSearch().SessionStart(context.Background(), protocolv2.FuzzyFileSearchSessionStartParams{})
	if err == nil {
		t.Fatal("FuzzyFileSearch().SessionStart succeeded with nil required roots after experimental opt-in")
	}
	if !strings.Contains(err.Error(), "encode fuzzyFileSearch/sessionStart params") ||
		!strings.Contains(err.Error(), "FuzzyFileSearchSessionStartParams.roots: nil is not allowed") {
		t.Fatalf("fuzzy file search sessionStart preflight error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodFuzzyFileSearchSessionStart) != nil {
		t.Fatal("fuzzyFileSearch/sessionStart was sent after preflight failure")
	}
}

func TestProtocolFacadeRejectsExperimentalFieldBeforeWriteUnlessOptedIn(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = root.Threads().Start(context.Background(), protocolv2.ThreadStartParams{
		ExperimentalRawEvents: Bool(true),
	})
	if err == nil {
		t.Fatal("Threads().Start accepted experimentalRawEvents without opt-in")
	}
	if !strings.Contains(err.Error(), "experimental field thread/start.experimentalRawEvents") {
		t.Fatalf("experimental field error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadStart) != nil {
		t.Fatal("thread/start was sent after experimental field preflight failure")
	}
	_, err = root.Threads().Start(context.Background(), protocolv2.ThreadStartParams{
		ExperimentalRawEvents: Bool(false),
	})
	if err == nil {
		t.Fatal("Threads().Start accepted explicit experimentalRawEvents=false without opt-in")
	}
	if !strings.Contains(err.Error(), "experimental field thread/start.experimentalRawEvents") {
		t.Fatalf("explicit experimentalRawEvents=false error = %v", err)
	}
	_, err = root.Threads().Start(context.Background(), protocolv2.ThreadStartParams{
		MockExperimentalField: protocolv2.Null[string](),
	})
	if err == nil {
		t.Fatal("Threads().Start accepted mockExperimentalField without opt-in")
	}
	if !strings.Contains(err.Error(), "experimental field thread/start.mockExperimentalField") {
		t.Fatalf("mockExperimentalField error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadStart) != nil {
		t.Fatal("thread/start was sent after experimental field preflight failures")
	}
	experimentalCases := []struct {
		name   string
		method string
		field  string
		call   func() error
	}{
		{
			name:   "command exec permission profile",
			method: protocolv2.MethodCommandExec,
			field:  "command/exec.permissionProfile",
			call: func() error {
				_, err := root.Commands().Exec(context.Background(), protocolv2.CommandExecParams{
					Command:           []string{"echo", "ok"},
					PermissionProfile: protocolv2.Null[string](),
				})
				return err
			},
		},
		{
			name:   "thread fork exclude turns",
			method: protocolv2.MethodThreadFork,
			field:  "thread/fork.excludeTurns",
			call: func() error {
				_, err := root.Threads().Fork(context.Background(), protocolv2.ThreadForkParams{
					ExcludeTurns: Bool(true),
					ThreadID:     "thread-1",
				})
				return err
			},
		},
		{
			name:   "thread resume history",
			method: protocolv2.MethodThreadResume,
			field:  "thread/resume.history",
			call: func() error {
				_, err := root.Threads().Resume(context.Background(), protocolv2.ThreadResumeParams{
					History:  protocolv2.Null[[]protocolv2.ResponseItem](),
					ThreadID: "thread-1",
				})
				return err
			},
		},
		{
			name:   "turn start responses api metadata",
			method: protocolv2.MethodTurnStart,
			field:  "turn/start.responsesapiClientMetadata",
			call: func() error {
				_, err := root.Turns().Start(context.Background(), protocolv2.TurnStartParams{
					Input: []protocolv2.UserInput{
						protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "hello"}),
					},
					ResponsesapiClientMetadata: protocolv2.Value(map[string]string{"trace": "turn-start"}),
					ThreadID:                   "thread-1",
				})
				return err
			},
		},
		{
			name:   "turn steer responses api metadata",
			method: protocolv2.MethodTurnSteer,
			field:  "turn/steer.responsesapiClientMetadata",
			call: func() error {
				_, err := root.Turns().Steer(context.Background(), protocolv2.TurnSteerParams{
					ExpectedTurnID: "turn-1",
					Input: []protocolv2.UserInput{
						protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "continue"}),
					},
					ResponsesapiClientMetadata: protocolv2.Value(map[string]string{"trace": "turn-steer"}),
					ThreadID:                   "thread-1",
				})
				return err
			},
		},
	}
	for _, tc := range experimentalCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil {
				t.Fatalf("%s accepted without opt-in", tc.field)
			}
			if !strings.Contains(err.Error(), "experimental field "+tc.field) {
				t.Fatalf("%s error = %v", tc.field, err)
			}
			if firstRecord(readRecords(t, record), "recv", tc.method) != nil {
				t.Fatalf("%s was sent after experimental field preflight failure", tc.method)
			}
		})
	}
	_ = root.Close()

	record = tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err = New(ClientOptions{
		CWD:          t.TempDir(),
		Command:      fakeCommand("facade"),
		Capabilities: ClientCapabilities{ExperimentalAPI: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := root.Threads().Start(context.Background(), protocolv2.ThreadStartParams{
		ExperimentalRawEvents: Bool(true),
	}); err != nil {
		t.Fatalf("Threads().Start rejected experimentalRawEvents after opt-in: %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadStart) == nil {
		t.Fatal("thread/start was not sent after experimental opt-in")
	}
	if _, err := root.Turns().Steer(context.Background(), protocolv2.TurnSteerParams{
		ExpectedTurnID: "turn-1",
		Input: []protocolv2.UserInput{
			protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "continue"}),
		},
		ResponsesapiClientMetadata: protocolv2.Value(map[string]string{"trace": "turn-steer"}),
		ThreadID:                   "thread-1",
	}); err != nil {
		t.Fatalf("Turns().Steer rejected responsesapiClientMetadata after opt-in: %v", err)
	}
	turnSteer := firstRecord(readRecords(t, record), "recv", protocolv2.MethodTurnSteer)
	if turnSteer == nil {
		t.Fatal("turn/steer was not sent after experimental opt-in")
	}
	turnSteerParams := turnSteer["params"].(map[string]any)
	turnSteerMetadata := turnSteerParams["responsesapiClientMetadata"].(map[string]any)
	if turnSteerMetadata["trace"] != "turn-steer" {
		t.Fatalf("turn/steer experimental metadata params = %#v", turnSteerParams)
	}
}

func TestProtocolFacadeRejectsMalformedTypedResponse(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("thread-start-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	_, err = root.Threads().Start(context.Background(), protocolv2.ThreadStartParams{})
	if err == nil {
		t.Fatal("Threads().Start accepted malformed thread/start response")
	}
	if !strings.Contains(err.Error(), "decode thread/start response") ||
		!strings.Contains(err.Error(), "ThreadStartResponse.approvalPolicy") {
		t.Fatalf("malformed facade response error = %v", err)
	}
}

func TestThreadProtocolFamilyFacadesRejectMalformedTypedResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("thread-list-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = root.Threads().List(context.Background(), protocolv2.ThreadListParams{})
	if err == nil {
		t.Fatal("Threads().List accepted malformed thread/list response")
	}
	if !strings.Contains(err.Error(), "decode thread/list response") ||
		!strings.Contains(err.Error(), "ThreadListResponse.data") {
		t.Fatalf("malformed thread/list response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadList) == nil {
		t.Fatal("thread/list was not sent before malformed response decode")
	}
	_ = root.Close()

	record = tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err = New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("thread-unsubscribe-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	_, err = root.Threads().Unsubscribe(context.Background(), protocolv2.ThreadUnsubscribeParams{ThreadID: "thread-1"})
	if err == nil {
		t.Fatal("Threads().Unsubscribe accepted malformed thread/unsubscribe response")
	}
	if !strings.Contains(err.Error(), "decode thread/unsubscribe response") ||
		!strings.Contains(err.Error(), `invalid ThreadUnsubscribeStatus enum value "bogus"`) {
		t.Fatalf("malformed thread/unsubscribe response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadUnsubscribe) == nil {
		t.Fatal("thread/unsubscribe was not sent before malformed response decode")
	}
}

func TestTurnProtocolFamilyFacadesRejectMalformedTypedResponses(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("turn-interrupt-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	_, err = root.Turns().Interrupt(context.Background(), protocolv2.TurnInterruptParams{
		ThreadID: "thread-1",
		TurnID:   "turn-1",
	})
	if err == nil {
		t.Fatal("Turns().Interrupt accepted malformed turn/interrupt response")
	}
	if !strings.Contains(err.Error(), "decode turn/interrupt response") ||
		!strings.Contains(err.Error(), `decode TurnInterruptResponse: unknown field "extra"`) {
		t.Fatalf("malformed turn/interrupt response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodTurnInterrupt) == nil {
		t.Fatal("turn/interrupt was not sent before malformed response decode")
	}
	_ = root.Close()

	record = tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err = New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("turn-steer-malformed-response")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	_, err = root.Turns().Steer(context.Background(), protocolv2.TurnSteerParams{
		ExpectedTurnID: "turn-1",
		Input:          []protocolv2.UserInput{protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "continue"})},
		ThreadID:       "thread-1",
	})
	if err == nil {
		t.Fatal("Turns().Steer accepted malformed turn/steer response")
	}
	if !strings.Contains(err.Error(), "decode turn/steer response") ||
		!strings.Contains(err.Error(), "TurnSteerResponse.turnId") {
		t.Fatalf("malformed turn/steer response error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodTurnSteer) == nil {
		t.Fatal("turn/steer was not sent before malformed response decode")
	}
}

func TestProtocolFacadeMethodsReturnErrClientClosed(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("facade")})
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := root.Threads().Start(context.Background(), protocolv2.ThreadStartParams{}); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("closed Threads().Start error = %v, want ErrClientClosed", err)
	}
	if _, err := root.Turns().Start(context.Background(), protocolv2.TurnStartParams{
		ThreadID: "thread-1",
		Input:    []protocolv2.UserInput{protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "closed"})},
	}); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("closed Turns().Start error = %v, want ErrClientClosed", err)
	}
	if _, err := root.Turns().Interrupt(context.Background(), protocolv2.TurnInterruptParams{
		ThreadID: "thread-1",
		TurnID:   "turn-1",
	}); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("closed Turns().Interrupt error = %v, want ErrClientClosed", err)
	}
	if _, err := root.Turns().Steer(context.Background(), protocolv2.TurnSteerParams{
		ExpectedTurnID: "turn-1",
		Input:          []protocolv2.UserInput{protocolv2.NewUserInputText(protocolv2.UserInputText{Text: "closed"})},
		ThreadID:       "thread-1",
	}); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("closed Turns().Steer error = %v, want ErrClientClosed", err)
	}
}

func TestThreadClientOptionsDefaultsStartThreadRequest(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	defaultEphemeral := true
	client := root.ThreadClient(ThreadClientOptions{
		DefaultModel:             "default-model",
		DefaultCWD:               "/workspace/thread",
		DefaultEffort:            ReasoningEffortHigh,
		DefaultApprovalPolicy:    ApprovalPolicyNever,
		DefaultApprovalsReviewer: ApprovalsReviewerAutoReview,
		DefaultEphemeral:         &defaultEphemeral,
	})
	defer root.Close()

	if _, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("defaults")}); err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	records := readRecords(t, record)
	start := firstRecord(records, "recv", "thread/start")["params"].(map[string]any)
	if start["model"] != "default-model" || start["cwd"] != "/workspace/thread" || start["approvalPolicy"] != "never" || start["approvalsReviewer"] != "auto_review" || start["ephemeral"] != true {
		t.Fatalf("thread/start params = %#v", start)
	}
	turnStart := firstRecord(records, "recv", "turn/start")["params"].(map[string]any)
	if turnStart["cwd"] != "/workspace/thread" || turnStart["effort"] != "high" {
		t.Fatalf("turn/start params = %#v", turnStart)
	}
}

func TestThreadClientStartRequestOverridesDefaults(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	client := root.ThreadClient(ThreadClientOptions{
		DefaultModel:             "default-model",
		DefaultCWD:               "/workspace/default",
		DefaultEffort:            ReasoningEffortLow,
		DefaultApprovalPolicy:    ApprovalPolicyNever,
		DefaultApprovalsReviewer: ApprovalsReviewerAutoReview,
		DefaultEphemeral:         Bool(true),
	})
	defer root.Close()

	if _, err := client.StartThread(context.Background(), StartThreadRequest{
		Input:             Text("overrides"),
		Model:             "request-model",
		CWD:               "/workspace/request",
		Effort:            ReasoningEffortHigh,
		ApprovalPolicy:    ApprovalPolicyOnRequest,
		ApprovalsReviewer: ApprovalsReviewerUser,
		Ephemeral:         Bool(false),
	}); err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	records := readRecords(t, record)
	start := firstRecord(records, "recv", protocolv2.MethodThreadStart)["params"].(map[string]any)
	if start["model"] != "request-model" || start["cwd"] != "/workspace/request" || start["approvalPolicy"] != "on-request" || start["approvalsReviewer"] != "user" || start["ephemeral"] != false {
		t.Fatalf("thread/start params = %#v", start)
	}
	turnStart := firstRecord(records, "recv", protocolv2.MethodTurnStart)["params"].(map[string]any)
	if turnStart["cwd"] != "/workspace/request" || turnStart["effort"] != "high" {
		t.Fatalf("turn/start params = %#v", turnStart)
	}
}

func TestThreadClientOptionsDefaultsResumeAndForkRequests(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	client := root.ThreadClient(ThreadClientOptions{
		DefaultModel:             "default-model",
		DefaultCWD:               "/workspace/thread",
		DefaultApprovalPolicy:    ApprovalPolicyNever,
		DefaultApprovalsReviewer: ApprovalsReviewerGuardianSubagent,
		DefaultEphemeral:         Bool(true),
	})
	defer root.Close()

	result, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("start")})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}
	if _, err := client.ResumeThread(context.Background(), ResumeThreadRequest{ThreadID: result.ThreadID, Input: Text("resume")}); err != nil {
		t.Fatalf("ResumeThread returned error: %v", err)
	}
	if _, err := client.ForkThread(context.Background(), ForkThreadRequest{ParentThreadID: result.ThreadID}); err != nil {
		t.Fatalf("ForkThread returned error: %v", err)
	}

	records := readRecords(t, record)
	resume := firstRecord(records, "recv", protocolv2.MethodThreadResume)["params"].(map[string]any)
	if resume["model"] != "default-model" || resume["cwd"] != "/workspace/thread" || resume["approvalPolicy"] != "never" || resume["approvalsReviewer"] != "guardian_subagent" {
		t.Fatalf("thread/resume params = %#v", resume)
	}
	turnStarts := recordsByMethod(records, "recv", protocolv2.MethodTurnStart)
	if turnStarts[1]["params"].(map[string]any)["cwd"] != "/workspace/thread" {
		t.Fatalf("resume turn/start params = %#v", turnStarts[1]["params"])
	}
	fork := firstRecord(records, "recv", protocolv2.MethodThreadFork)["params"].(map[string]any)
	if fork["model"] != "default-model" || fork["cwd"] != "/workspace/thread" || fork["approvalPolicy"] != "never" || fork["approvalsReviewer"] != "guardian_subagent" || fork["ephemeral"] != true {
		t.Fatalf("thread/fork params = %#v", fork)
	}
}

func TestThreadClientRejectsInvalidPolicyDefaultsBeforeThreadStart(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options ThreadClientOptions
		wantErr string
	}{
		{
			name: "approval policy",
			options: ThreadClientOptions{
				DefaultModel:          "client-model",
				DefaultApprovalPolicy: ApprovalPolicy("root"),
			},
			wantErr: `unsupported ApprovalPolicy "root"`,
		},
		{
			name: "approvals reviewer",
			options: ThreadClientOptions{
				DefaultModel:             "client-model",
				DefaultApprovalsReviewer: ApprovalsReviewer("committee"),
			},
			wantErr: `unsupported ApprovalsReviewer "committee"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := tempRecord(t)
			t.Setenv("CODEXSDK_FAKE_RECORD", record)
			root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()

			client := root.ThreadClient(tc.options)
			_, err = client.StartThread(context.Background(), StartThreadRequest{Input: Text("invalid default")})
			if err == nil {
				t.Fatalf("StartThread accepted invalid default %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("invalid default %s error = %v", tc.name, err)
			}
			if firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadStart) != nil {
				t.Fatalf("thread/start was sent after invalid default %s preflight failure", tc.name)
			}
		})
	}
}

func TestThreadClientZeroValueOptionsDoNotSetPolicyDefaults(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	client := root.ThreadClient(ThreadClientOptions{})
	if _, err := client.StartThread(context.Background(), StartThreadRequest{
		Input: Text("zero"),
		Model: "request-model",
	}); err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	start := firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadStart)["params"].(map[string]any)
	if _, ok := start["approvalPolicy"]; ok {
		t.Fatalf("zero-value options set approval policy: %#v", start)
	}
	if _, ok := start["approvalsReviewer"]; ok {
		t.Fatalf("zero-value options set approvals reviewer: %#v", start)
	}
	if _, ok := start["ephemeral"]; ok {
		t.Fatalf("zero-value options set ephemeral: %#v", start)
	}
}

func TestThreadClientStartRequiresRequestOrDefaultModel(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	client := root.ThreadClient(ThreadClientOptions{})
	_, err = client.StartThread(context.Background(), StartThreadRequest{Input: Text("missing model")})
	if err == nil {
		t.Fatal("StartThread succeeded without request model or default model")
	}
	if !strings.Contains(err.Error(), "StartThreadRequest.Model or ThreadClientOptions.DefaultModel is required") {
		t.Fatalf("missing model error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", "thread/start") != nil {
		t.Fatal("thread/start was sent before model preflight")
	}
}

func TestThreadClientRejectsUnsupportedInputTypeBeforeThreadStart(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	client := root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"})
	_, err = client.StartThread(context.Background(), StartThreadRequest{
		Input: []InputItem{{Type: "custom", Text: "raw"}},
	})
	if err == nil {
		t.Fatal("StartThread accepted unsupported InputItem type")
	}
	if !strings.Contains(err.Error(), `unsupported InputItem[0].Type "custom"`) {
		t.Fatalf("unsupported input error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadStart) != nil {
		t.Fatal("thread/start was sent after unsupported input preflight failure")
	}
}

func TestThreadClientRejectsBlankFileInputBeforeThreadStart(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	client := root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"})
	_, err = client.StartThread(context.Background(), StartThreadRequest{
		Input: []InputItem{
			{Type: InputItemText, Text: "read this"},
			{Type: InputItemFile, Path: "  "},
		},
	})
	if err == nil {
		t.Fatal("StartThread accepted blank file input path")
	}
	if !strings.Contains(err.Error(), "InputItem[1].Path is required for file input") {
		t.Fatalf("blank file input error = %v", err)
	}
	if firstRecord(readRecords(t, record), "recv", protocolv2.MethodThreadStart) != nil {
		t.Fatal("thread/start was sent after blank file input preflight failure")
	}
}

func TestThreadClientRejectsMalformedTypedLifecycleResponses(t *testing.T) {
	for _, tc := range []struct {
		name           string
		mode           string
		wantErr        string
		unwantedMethod string
	}{
		{
			name:           "thread start",
			mode:           "thread-start-malformed-response",
			wantErr:        "decode thread/start response",
			unwantedMethod: protocolv2.MethodTurnStart,
		},
		{
			name:    "turn start",
			mode:    "turn-start-malformed-response",
			wantErr: "decode turn/start response",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := tempRecord(t)
			t.Setenv("CODEXSDK_FAKE_RECORD", record)
			root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand(tc.mode)})
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()

			client := root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"})
			_, err = client.StartThread(context.Background(), StartThreadRequest{Input: Text("typed lifecycle")})
			if err == nil {
				t.Fatalf("StartThread accepted malformed %s response", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("malformed %s error = %v", tc.name, err)
			}
			if tc.unwantedMethod != "" && firstRecord(readRecords(t, record), "recv", tc.unwantedMethod) != nil {
				t.Fatalf("%s was sent after %s response validation failure", tc.unwantedMethod, tc.name)
			}
		})
	}
}

func TestThreadClientRejectsMalformedKnownNotificationBeforeStreamDispatch(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mode    string
		want    string
		wantSub string
	}{
		{
			name:    "agent delta",
			mode:    "malformed-notification",
			want:    "decode item/agentMessage/delta notification",
			wantSub: "AgentMessageDeltaNotification.itemId",
		},
		{
			name:    "token usage",
			mode:    "malformed-token-usage-notification",
			want:    "decode thread/tokenUsage/updated notification",
			wantSub: "ThreadTokenUsage.total",
		},
		{
			name:    "turn plan",
			mode:    "malformed-turn-plan-notification",
			want:    "decode turn/plan/updated notification",
			wantSub: "TurnPlanUpdatedNotification.plan",
		},
		{
			name:    "hook started",
			mode:    "malformed-hook-started-notification",
			want:    "decode hook/started notification",
			wantSub: "HookStartedNotification.run",
		},
		{
			name:    "hook completed",
			mode:    "malformed-hook-completed-notification",
			want:    "decode hook/completed notification",
			wantSub: "HookCompletedNotification.threadId",
		},
		{
			name:    "thread goal updated",
			mode:    "malformed-thread-goal-updated-notification",
			want:    "decode thread/goal/updated notification",
			wantSub: "ThreadGoalUpdatedNotification.goal",
		},
		{
			name:    "realtime output audio",
			mode:    "malformed-realtime-output-audio-notification",
			want:    "decode thread/realtime/outputAudio/delta notification",
			wantSub: "ThreadRealtimeOutputAudioDeltaNotification.audio",
		},
		{
			name:    "guardian review started",
			mode:    "malformed-guardian-review-started-notification",
			want:    "decode item/autoApprovalReview/started notification",
			wantSub: "ItemGuardianApprovalReviewStartedNotification.action",
		},
		{
			name:    "guardian review completed",
			mode:    "malformed-guardian-review-completed-notification",
			want:    "decode item/autoApprovalReview/completed notification",
			wantSub: "ItemGuardianApprovalReviewCompletedNotification.decisionSource",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			record := tempRecord(t)
			t.Setenv("CODEXSDK_FAKE_RECORD", record)
			root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand(tc.mode)})
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()

			client := root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"})
			_, err = client.StartThread(context.Background(), StartThreadRequest{Input: Text("malformed notification")})
			if err == nil {
				t.Fatalf("StartThread accepted malformed %s notification", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("malformed notification error = %v", err)
			}
		})
	}
}

func TestThreadClientValidatesBackgroundNotificationsBeforeTypedIgnore(t *testing.T) {
	for _, tc := range []struct {
		name    string
		mode    string
		wantErr string
	}{
		{
			name:    "unknown notification method",
			mode:    "unknown-notification",
			wantErr: `decode ServerNotification.method: unknown variant "upstream/newNotification"`,
		},
		{
			name:    "malformed background notification",
			mode:    "malformed-background-notification",
			wantErr: `decode ServerNotification.params: decode SkillsChangedNotification: unknown field "extra"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := newFakeClient(t, tc.mode, nil)
			defer client.Close()

			_, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("notification guard")})
			if err == nil {
				t.Fatalf("StartThread accepted %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("%s error = %v", tc.name, err)
			}
		})
	}
}

func TestThreadClientIgnoresValidatedBackgroundNotification(t *testing.T) {
	client := newFakeClient(t, "background-notification", nil)
	defer client.Close()

	result, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("background notification")})
	if err != nil {
		t.Fatal(err)
	}
	if result.FinalResponse != "final-turn-1" {
		t.Fatalf("result = %#v", result)
	}
}

func TestRequestMappingSteeringForkAndAggregateInputStats(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	client := root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"})
	defer root.Close()

	result, err := client.StartThread(context.Background(), StartThreadRequest{
		Input:             TextAndFiles("hello", []string{"/tmp/a.txt", "/tmp/b.txt"}),
		Ephemeral:         Bool(true),
		Effort:            ReasoningEffortHigh,
		ApprovalPolicy:    ApprovalPolicyNever,
		ApprovalsReviewer: ApprovalsReviewerAutoReview,
	})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}
	if result.FinalResponse != "final-turn-1" || result.InputStats.ItemsCount != 3 || result.InputStats.AttachmentCount != 2 || result.InputStats.TextBytes != 5 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.InputStats.InputItemsHash == "" {
		t.Fatal("InputItemsHash is empty")
	}

	if _, err := client.ResumeThread(context.Background(), ResumeThreadRequest{ThreadID: result.ThreadID, Input: Text("resume")}); err != nil {
		t.Fatalf("ResumeThread returned error: %v", err)
	}
	if _, err := client.ResumeThread(context.Background(), ResumeThreadRequest{ThreadID: result.ThreadID, Input: Text("resume"), Model: "resume-model"}); err != nil {
		t.Fatalf("ResumeThread with model returned error: %v", err)
	}
	if _, err := client.ForkThread(context.Background(), ForkThreadRequest{
		ParentThreadID:    result.ThreadID,
		Ephemeral:         Bool(false),
		ApprovalPolicy:    ApprovalPolicyOnRequest,
		ApprovalsReviewer: ApprovalsReviewerUser,
	}); err != nil {
		t.Fatalf("ForkThread returned error: %v", err)
	}

	records := readRecords(t, record)
	start := firstRecord(records, "recv", "thread/start")["params"].(map[string]any)
	if start["model"] != "client-model" || start["ephemeral"] != true || start["approvalPolicy"] != "never" || start["approvalsReviewer"] != "auto_review" {
		t.Fatalf("thread/start params = %#v", start)
	}
	turnStart := firstRecord(records, "recv", "turn/start")["params"].(map[string]any)
	if turnStart["effort"] != "high" {
		t.Fatalf("turn/start effort = %#v", turnStart)
	}
	input := turnStart["input"].([]any)
	if len(input) != 3 {
		t.Fatalf("turn/start input = %#v", input)
	}
	textItem := input[0].(map[string]any)
	if textItem["type"] != "text" || textItem["text"] != "hello" {
		t.Fatalf("text input item = %#v", textItem)
	}
	if _, ok := textItem["text_elements"].([]any); !ok {
		t.Fatalf("text input item missing text_elements: %#v", textItem)
	}
	mention := input[1].(map[string]any)
	if mention["type"] != "mention" || mention["name"] != "a.txt" || mention["path"] != "/tmp/a.txt" {
		t.Fatalf("mention input item = %#v", mention)
	}
	resumes := recordsByMethod(records, "recv", "thread/resume")
	if resumes[0]["params"].(map[string]any)["model"] != "client-model" {
		t.Fatalf("resume without model did not use client default: %#v", resumes[0]["params"])
	}
	if resumes[1]["params"].(map[string]any)["model"] != "resume-model" {
		t.Fatalf("resume model params = %#v", resumes[1]["params"])
	}
	fork := firstRecord(records, "recv", "thread/fork")["params"].(map[string]any)
	if fork["approvalPolicy"] != "on-request" || fork["approvalsReviewer"] != "user" {
		t.Fatalf("fork steering params = %#v", fork)
	}
	if len(recordsByMethod(records, "recv", "thread/fork")) != 1 {
		t.Fatal("expected exactly one thread/fork")
	}
}

func TestStreamingCompletedResultAndFinalAPIDrainsSamePath(t *testing.T) {
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	client := root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"})
	defer root.Close()

	stream, err := client.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("stream")})
	if err != nil {
		t.Fatal(err)
	}
	var kinds []ThreadEventKind
	for stream.Next(context.Background()) {
		event := stream.Event()
		kinds = append(kinds, event.Kind)
		if event.Kind == ThreadEventCompleted && (event.Result == nil || event.Result.FinalResponse == "") {
			t.Fatalf("completed event missing result: %#v", event)
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream err = %v", err)
	}
	if got, want := kinds, []ThreadEventKind{ThreadEventStarted, ThreadEventUsage, ThreadEventCompleted}; !reflect.DeepEqual(got, want) {
		t.Fatalf("event kinds = %#v, want %#v", got, want)
	}
	if result, ok := stream.Result(); !ok || result.FinalResponse != "final-turn-1" {
		t.Fatalf("stream result = %#v ok=%v", result, ok)
	}

	result, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("final")})
	if err != nil {
		t.Fatal(err)
	}
	if result.FinalResponse != "final-turn-2" {
		t.Fatalf("final API result = %#v", result)
	}
}

func TestFinalResponseDoesNotComeFromOutputDelta(t *testing.T) {
	client := newFakeClient(t, "delta-without-final", nil)
	defer client.Close()

	_, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("delta is not final")})
	if err == nil {
		t.Fatal("StartThread succeeded without completed final_answer item")
	}
	if !strings.Contains(err.Error(), "codexsdk: turn completed without final_answer agent message") {
		t.Fatalf("missing final_answer error = %v", err)
	}
}

func TestTurnCompletedNonCompletedStatusDoesNotProduceFinalResult(t *testing.T) {
	client := newFakeClient(t, "turn-completed-in-progress", nil)
	defer client.Close()

	_, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("non terminal")})
	if err == nil {
		t.Fatal("StartThread succeeded on turn/completed with inProgress status")
	}
	if !strings.Contains(err.Error(), `codexsdk: turn/completed received non-completed turn status "inProgress"`) {
		t.Fatalf("non-completed status error = %v", err)
	}
}

func TestThreadStreamEmitsDiagnosticForNilHandlerFailClosedServerRequest(t *testing.T) {
	client := newFakeClient(t, "approval", nil)
	defer client.Close()

	stream, err := client.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("approval diagnostic")})
	if err != nil {
		t.Fatal(err)
	}
	var diagnostic *DiagnosticRef
	var completed bool
	for stream.Next(context.Background()) {
		event := stream.Event()
		switch event.Kind {
		case ThreadEventDiagnostic:
			if event.ThreadID != "thread-1" || event.TurnID != "turn-1" {
				t.Fatalf("diagnostic event identity = %#v", event)
			}
			diagnostic = event.Diagnostic
		case ThreadEventCompleted:
			completed = true
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream err = %v", err)
	}
	if !completed {
		t.Fatal("stream did not complete after nil-handler fail-closed response")
	}
	if diagnostic == nil {
		t.Fatal("missing diagnostic event for nil-handler fail-closed server request")
	}
	if diagnostic.Kind != "server_request_fail_closed" || diagnostic.ID != "server-approval-1" {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
}

func TestOutputSchemaMappingStartResumeAndStreaming(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	client := root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"})
	defer root.Close()

	startFinalSchema := mustOutputSchema(t, `{"type":"object","properties":{"startFinal":{"type":"string"}},"required":["startFinal"],"additionalProperties":false}`)
	startStreamSchema := mustOutputSchema(t, `{"type":"object","properties":{"startStream":{"type":"boolean"}},"required":["startStream"],"additionalProperties":false}`)
	resumeFinalSchema := mustOutputSchema(t, `{"type":"object","properties":{"resumeFinal":{"type":"number"}},"required":["resumeFinal"],"additionalProperties":false}`)
	resumeStreamSchema := mustOutputSchema(t, `{"type":"object","properties":{"resumeStream":{"type":"array","items":{"type":"string"}}},"required":["resumeStream"],"additionalProperties":false}`)

	result, err := client.StartThread(context.Background(), StartThreadRequest{
		Input:        Text("start final"),
		OutputSchema: startFinalSchema,
	})
	if err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}

	startStream, err := client.StartThreadStream(context.Background(), StartThreadRequest{
		Input:        Text("start stream"),
		OutputSchema: startStreamSchema,
	})
	if err != nil {
		t.Fatalf("StartThreadStream returned error: %v", err)
	}
	for startStream.Next(context.Background()) {
	}
	if err := startStream.Err(); err != nil {
		t.Fatalf("start stream err = %v", err)
	}

	if _, err := client.ResumeThread(context.Background(), ResumeThreadRequest{
		ThreadID:     result.ThreadID,
		Input:        Text("resume final"),
		OutputSchema: resumeFinalSchema,
	}); err != nil {
		t.Fatalf("ResumeThread returned error: %v", err)
	}

	resumeStream, err := client.ResumeThreadStream(context.Background(), ResumeThreadRequest{
		ThreadID:     result.ThreadID,
		Input:        Text("resume stream"),
		OutputSchema: resumeStreamSchema,
	})
	if err != nil {
		t.Fatalf("ResumeThreadStream returned error: %v", err)
	}
	for resumeStream.Next(context.Background()) {
	}
	if err := resumeStream.Err(); err != nil {
		t.Fatalf("resume stream err = %v", err)
	}

	turnStarts := recordsByMethod(readRecords(t, record), "recv", "turn/start")
	if len(turnStarts) != 4 {
		t.Fatalf("turn/start records = %d, want 4: %#v", len(turnStarts), turnStarts)
	}
	assertJSONEqual(t, turnStarts[0]["params"].(map[string]any)["outputSchema"], `{"type":"object","properties":{"startFinal":{"type":"string"}},"required":["startFinal"],"additionalProperties":false}`)
	assertJSONEqual(t, turnStarts[1]["params"].(map[string]any)["outputSchema"], `{"type":"object","properties":{"startStream":{"type":"boolean"}},"required":["startStream"],"additionalProperties":false}`)
	assertJSONEqual(t, turnStarts[2]["params"].(map[string]any)["outputSchema"], `{"type":"object","properties":{"resumeFinal":{"type":"number"}},"required":["resumeFinal"],"additionalProperties":false}`)
	assertJSONEqual(t, turnStarts[3]["params"].(map[string]any)["outputSchema"], `{"type":"object","properties":{"resumeStream":{"type":"array","items":{"type":"string"}}},"required":["resumeStream"],"additionalProperties":false}`)
}

func TestEmptyOutputSchemaOmitted(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("happy")})
	if err != nil {
		t.Fatal(err)
	}
	client := root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"})
	defer root.Close()

	if _, err := client.StartThread(context.Background(), StartThreadRequest{
		Input: Text("empty schema"),
	}); err != nil {
		t.Fatalf("StartThread returned error: %v", err)
	}
	turnStart := firstRecord(readRecords(t, record), "recv", "turn/start")["params"].(map[string]any)
	if _, ok := turnStart["outputSchema"]; ok {
		t.Fatalf("empty outputSchema was sent: %#v", turnStart)
	}
}

func TestPublicThreadRequestsUseTypedOutputSchema(t *testing.T) {
	want := reflect.TypeOf(protocolv2.OutputSchema{})
	for _, typ := range []reflect.Type{
		reflect.TypeOf(StartThreadRequest{}),
		reflect.TypeOf(ResumeThreadRequest{}),
	} {
		field, ok := typ.FieldByName("OutputSchema")
		if !ok {
			t.Fatalf("%s missing OutputSchema field", typ.Name())
		}
		if field.Type != want {
			t.Fatalf("%s.OutputSchema type = %s, want %s", typ.Name(), field.Type, want)
		}
	}
}

func TestInvalidOutputSchemaFailsAtTypedParseBoundary(t *testing.T) {
	_, err := protocolv2.OutputSchemaFromJSON([]byte(`{"type":`))
	if err == nil {
		t.Fatal("OutputSchemaFromJSON accepted invalid outputSchema")
	}
	if !strings.Contains(err.Error(), "EOF") {
		t.Fatalf("invalid outputSchema error = %v", err)
	}
}

func TestNativeNotificationsAndLongLines(t *testing.T) {
	client := newFakeClient(t, "native-notifications", nil)
	defer client.Close()
	stream, err := client.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("native")})
	if err != nil {
		t.Fatal(err)
	}
	var delta, warningMessage, fromModel, toModel, configSummary, configDetails, configPath string
	var warningWillRetry bool
	var verifications []string
	var configTurnID *string
	for stream.Next(context.Background()) {
		event := stream.Event()
		switch event.Kind {
		case ThreadEventOutputDelta:
			delta += event.OutputDelta
		case ThreadEventTurnWarning:
			warningMessage = event.TurnWarning.Message
			warningWillRetry = event.TurnWarning.WillRetry
		case ThreadEventModelRerouted:
			fromModel = event.Model.FromModel
			toModel = event.Model.ToModel
		case ThreadEventModelVerification:
			verifications = append([]string(nil), event.Model.Verifications...)
		case ThreadEventConfigWarning:
			turnID := event.TurnID
			configTurnID = &turnID
			configSummary = event.Warning.Summary
			configDetails = event.Warning.Details
			configPath = event.Warning.Path
		}
	}
	if err := stream.Err(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(delta, "native delta") || warningMessage != "native warning" || !warningWillRetry {
		t.Fatalf("native warning/delta delta=%q warning=%q willRetry=%v", delta, warningMessage, warningWillRetry)
	}
	if fromModel != "from-model" || toModel != "to-model" {
		t.Fatalf("native reroute from=%q to=%q", fromModel, toModel)
	}
	if !reflect.DeepEqual(verifications, []string{string(protocolv2.ModelVerificationTrustedAccessForCyber)}) {
		t.Fatalf("native verifications = %#v", verifications)
	}
	if configTurnID == nil || *configTurnID != "" || configSummary != "native config summary" || configDetails != "native config details" || configPath != "/tmp/config.toml" {
		t.Fatalf("native config turnID=%v summary=%q details=%q path=%q", configTurnID, configSummary, configDetails, configPath)
	}

	longLine := newFakeClient(t, "long-line", nil)
	defer longLine.Close()
	result, err := longLine.StartThread(context.Background(), StartThreadRequest{Input: Text("long")})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.FinalResponse) <= 64*1024 {
		t.Fatalf("final response length = %d, want over scanner limit", len(result.FinalResponse))
	}
}

func TestAttachStreamDrainsPendingStateTogether(t *testing.T) {
	turnErr := errors.New("turn pending")
	globalErr := errors.New("global pending")
	c := &client{
		streams: map[string]map[*threadStreamState]struct{}{},
		pendingEvents: map[string][]rpcNotification{"turn-1": {
			{method: "item/agentMessage/delta", params: map[string]any{"threadId": "thread-1", "turnId": "turn-1", "delta": "turn"}},
		}},
		pendingErrors: map[string]error{"turn-1": turnErr},
		pendingServer: map[string][]rpcServerRequest{"turn-1": {
			{id: "server-1", method: "item/commandExecution/requestApproval", params: map[string]any{"threadId": "thread-1", "turnId": "turn-1"}},
		}},
		pendingGlobal: globalErr,
		pendingThreadEvents: map[string][]rpcNotification{"thread-1": {
			{method: "model/rerouted", params: map[string]any{"threadId": "thread-1", "fromModel": "a", "toModel": "b"}},
		}},
		pendingGlobalEvents: []rpcNotification{
			{method: "configWarning", params: map[string]any{"summary": "global"}},
		},
	}
	stream := newThreadStream(c, "thread-1").state

	notifications, serverRequests, pendingErr := c.attachStreamAndDrainPending("turn-1", stream)

	if _, ok := c.streams["turn-1"][stream]; !ok {
		t.Fatal("stream was not attached")
	}
	if got := len(notifications); got != 3 {
		t.Fatalf("notifications drained = %d, want 3", got)
	}
	if got := len(serverRequests); got != 1 {
		t.Fatalf("server requests drained = %d, want 1", got)
	}
	if pendingErr == nil || !strings.Contains(pendingErr.Error(), turnErr.Error()) || !strings.Contains(pendingErr.Error(), globalErr.Error()) {
		t.Fatalf("pending error = %v, want joined turn and global errors", pendingErr)
	}
	if len(c.pendingGlobalEvents) != 0 || len(c.pendingThreadEvents) != 0 || len(c.pendingEvents) != 0 || len(c.pendingErrors) != 0 || len(c.pendingServer) != 0 || c.pendingGlobal != nil {
		t.Fatalf("pending state was not cleared: %#v %#v %#v %#v %#v %v", c.pendingGlobalEvents, c.pendingThreadEvents, c.pendingEvents, c.pendingErrors, c.pendingServer, c.pendingGlobal)
	}
}

func TestFailClosedServerRequestDoesNotInvokeHandler(t *testing.T) {
	handlerCalled := make(chan struct{}, 1)
	writer := &recordingWriteCloser{}
	c := &client{
		stdin:         writer,
		streams:       map[string]map[*threadStreamState]struct{}{},
		pendingErrors: map[string]error{},
		options: ClientOptions{LegacyServerRequestHandler: func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
			handlerCalled <- struct{}{}
			return LegacyServerRequestResponse{ApprovalDecision: ApprovalAccept}, nil
		}},
	}

	c.respondToServerRequestFailClosed("server-1", "item/commandExecution/requestApproval", fakeCommandApprovalParams("thread-1", "turn-1"))

	select {
	case <-handlerCalled:
		t.Fatal("fail-closed request invoked handler")
	default:
	}
	var response map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(writer.Bytes()), &response); err != nil {
		t.Fatalf("fail-closed response JSON = %q: %v", writer.String(), err)
	}
	result, _ := response["result"].(map[string]any)
	if response["id"] != "server-1" || result["decision"] != string(ApprovalDecline) {
		t.Fatalf("fail-closed response = %#v", response)
	}
}

func TestServerRequestApprovalFailClosedResponsesUseProtocolShapes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		params map[string]any
		assert func(t *testing.T, result map[string]any)
	}{
		{
			name:   "apply patch",
			method: protocolv2.MethodApplyPatchApproval,
			params: map[string]any{
				"callId":         "call-1",
				"conversationId": "thread-1",
				"fileChanges": map[string]any{
					"/repo/file.txt": map[string]any{"content": "new", "type": "add"},
				},
			},
			assert: func(t *testing.T, result map[string]any) {
				if result["decision"] != "denied" {
					t.Fatalf("applyPatchApproval fail-closed result = %#v", result)
				}
			},
		},
		{
			name:   "exec command",
			method: protocolv2.MethodExecCommandApproval,
			params: map[string]any{
				"callId":         "call-1",
				"command":        []any{"echo", "ok"},
				"conversationId": "thread-1",
				"cwd":            "/repo",
				"parsedCmd":      []any{},
			},
			assert: func(t *testing.T, result map[string]any) {
				if result["decision"] != "denied" {
					t.Fatalf("execCommandApproval fail-closed result = %#v", result)
				}
			},
		},
		{
			name:   "command execution",
			method: protocolv2.MethodItemCommandExecutionRequestApproval,
			params: fakeCommandApprovalParams("thread-1", "turn-1"),
			assert: func(t *testing.T, result map[string]any) {
				if result["decision"] != "decline" {
					t.Fatalf("command execution fail-closed result = %#v", result)
				}
			},
		},
		{
			name:   "file change",
			method: protocolv2.MethodItemFileChangeRequestApproval,
			params: map[string]any{
				"itemId":      "item-file",
				"startedAtMs": float64(123),
				"threadId":    "thread-1",
				"turnId":      "turn-1",
			},
			assert: func(t *testing.T, result map[string]any) {
				if result["decision"] != "decline" {
					t.Fatalf("file change fail-closed result = %#v", result)
				}
			},
		},
		{
			name:   "permissions",
			method: protocolv2.MethodItemPermissionsRequestApproval,
			params: map[string]any{
				"cwd":         "/repo",
				"itemId":      "item-permissions",
				"permissions": map[string]any{"network": map[string]any{"enabled": true}},
				"startedAtMs": float64(123),
				"threadId":    "thread-1",
				"turnId":      "turn-1",
			},
			assert: func(t *testing.T, result map[string]any) {
				if _, ok := result["decision"]; ok {
					t.Fatalf("permissions fail-closed result used decision field: %#v", result)
				}
				permissions, ok := result["permissions"].(map[string]any)
				if !ok || len(permissions) != 0 {
					t.Fatalf("permissions fail-closed result = %#v", result)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			writer := &recordingWriteCloser{}
			c := &client{stdin: writer}
			c.respondToServerRequestFailClosed("server-1", tc.method, tc.params)
			var response map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(writer.Bytes()), &response); err != nil {
				t.Fatalf("fail-closed response JSON = %q: %v", writer.String(), err)
			}
			if response["id"] != "server-1" {
				t.Fatalf("response id = %#v", response)
			}
			result, ok := response["result"].(map[string]any)
			if !ok {
				t.Fatalf("response missing result object: %#v", response)
			}
			tc.assert(t, result)
		})
	}
}

func TestServerRequestNonApprovalFailClosedResponsesUseProtocolShapes(t *testing.T) {
	tests := []struct {
		name   string
		method string
		params map[string]any
		assert func(t *testing.T, response map[string]any)
	}{
		{
			name:   "tool call",
			method: protocolv2.MethodItemToolCall,
			params: fakeDynamicToolCallParams("thread-1", "turn-tool"),
			assert: func(t *testing.T, response map[string]any) {
				result, ok := response["result"].(map[string]any)
				if !ok || result["success"] != false {
					t.Fatalf("tool call fail-closed result = %#v", response)
				}
				contentItems, ok := result["contentItems"].([]any)
				if !ok || len(contentItems) != 0 {
					t.Fatalf("tool call contentItems = %#v", response)
				}
			},
		},
		{
			name:   "tool user input",
			method: protocolv2.MethodItemToolRequestUserInput,
			params: fakeToolUserInputParams("thread-1", "turn-input"),
			assert: func(t *testing.T, response map[string]any) {
				result, ok := response["result"].(map[string]any)
				if !ok {
					t.Fatalf("user input fail-closed result = %#v", response)
				}
				answers, ok := result["answers"].(map[string]any)
				if !ok || len(answers) != 0 {
					t.Fatalf("user input answers = %#v", response)
				}
			},
		},
		{
			name:   "mcp elicitation",
			method: protocolv2.MethodMCPServerElicitationRequest,
			params: fakeMCPElicitationParams("thread-1", "turn-mcp"),
			assert: func(t *testing.T, response map[string]any) {
				result, ok := response["result"].(map[string]any)
				if !ok || result["action"] != "decline" {
					t.Fatalf("mcp elicitation fail-closed result = %#v", response)
				}
			},
		},
		{
			name:   "chatgpt auth refresh",
			method: protocolv2.MethodAccountChatGPTAuthTokensRefresh,
			params: fakeAuthRefreshParams(),
			assert: func(t *testing.T, response map[string]any) {
				if _, ok := response["result"]; ok {
					t.Fatalf("auth refresh fail-closed fabricated result: %#v", response)
				}
				errorObject, _ := response["error"].(map[string]any)
				if errorObject["code"] != float64(-32000) ||
					!strings.Contains(fmt.Sprint(errorObject["message"]), "requires typed ChatGPTAuthTokensRefresh response") {
					t.Fatalf("auth refresh fail-closed error = %#v", response)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			writer := &recordingWriteCloser{}
			c := &client{stdin: writer}
			c.respondToServerRequestFailClosed("server-1", tc.method, tc.params)
			var response map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(writer.Bytes()), &response); err != nil {
				t.Fatalf("fail-closed response JSON = %q: %v", writer.String(), err)
			}
			if response["id"] != "server-1" {
				t.Fatalf("response id = %#v", response)
			}
			tc.assert(t, response)
		})
	}
}

func TestServerRequestRejectsMalformedTypedApprovalBeforeHandler(t *testing.T) {
	handlerCalled := make(chan struct{}, 1)
	writer := &recordingWriteCloser{}
	c := &client{
		stdin:         writer,
		streams:       map[string]map[*threadStreamState]struct{}{},
		pendingErrors: map[string]error{},
		options: ClientOptions{LegacyServerRequestHandler: func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
			handlerCalled <- struct{}{}
			return LegacyServerRequestResponse{ApprovalDecision: ApprovalAccept}, nil
		}},
	}

	c.respondToServerRequest("server-1", protocolv2.MethodItemCommandExecutionRequestApproval, map[string]any{
		"itemId":   "item-1",
		"threadId": "thread-1",
		"turnId":   "turn-1",
	})

	select {
	case <-handlerCalled:
		t.Fatal("malformed approval request invoked handler")
	default:
	}
	var response map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(writer.Bytes()), &response); err != nil {
		t.Fatalf("malformed approval response JSON = %q: %v", writer.String(), err)
	}
	errorObject, _ := response["error"].(map[string]any)
	if response["id"] != "server-1" || errorObject["code"] != float64(-32602) ||
		!strings.Contains(fmt.Sprint(errorObject["message"]), "CommandExecutionRequestApprovalParams.startedAtMs") {
		t.Fatalf("malformed approval error response = %#v", response)
	}
}

func TestServerRequestHandlerTypedApprovalResponses(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		params   map[string]any
		response LegacyServerRequestResponse
		assert   func(t *testing.T, req ServerRequest, result map[string]any)
	}{
		{
			name:   "apply patch",
			method: protocolv2.MethodApplyPatchApproval,
			params: map[string]any{
				"callId":         "call-1",
				"conversationId": "thread-1",
				"fileChanges": map[string]any{
					"/repo/file.txt": map[string]any{"content": "old", "type": "delete"},
				},
			},
			response: LegacyServerRequestResponse{ApprovalDecision: ApprovalAccept},
			assert: func(t *testing.T, req ServerRequest, result map[string]any) {
				if req.Kind != ServerRequestApplyPatchApproval || req.ApplyPatchApproval == nil || req.ThreadID != "thread-1" || req.ItemID != "call-1" {
					t.Fatalf("applyPatchApproval request = %#v", req)
				}
				if result["decision"] != "approved" {
					t.Fatalf("applyPatchApproval response = %#v", result)
				}
			},
		},
		{
			name:   "exec command",
			method: protocolv2.MethodExecCommandApproval,
			params: map[string]any{
				"callId":         "call-1",
				"command":        []any{"echo", "ok"},
				"conversationId": "thread-1",
				"cwd":            "/repo",
				"parsedCmd":      []any{},
			},
			response: LegacyServerRequestResponse{ApprovalDecision: ApprovalAcceptForSession},
			assert: func(t *testing.T, req ServerRequest, result map[string]any) {
				if req.Kind != ServerRequestExecCommandApproval || req.ExecCommandApproval == nil || !reflect.DeepEqual(req.Approval.Command, []string{"echo", "ok"}) {
					t.Fatalf("execCommandApproval request = %#v", req)
				}
				if result["decision"] != "approved_for_session" {
					t.Fatalf("execCommandApproval response = %#v", result)
				}
			},
		},
		{
			name:     "command execution",
			method:   protocolv2.MethodItemCommandExecutionRequestApproval,
			params:   fakeCommandApprovalParams("thread-1", "turn-1"),
			response: LegacyServerRequestResponse{ApprovalDecision: ApprovalCancel},
			assert: func(t *testing.T, req ServerRequest, result map[string]any) {
				if req.Kind != ServerRequestCommandApproval || req.CommandExecutionApproval == nil || req.TurnID != "turn-1" {
					t.Fatalf("command execution request = %#v", req)
				}
				if result["decision"] != "cancel" {
					t.Fatalf("command execution response = %#v", result)
				}
			},
		},
		{
			name:   "file change",
			method: protocolv2.MethodItemFileChangeRequestApproval,
			params: map[string]any{
				"itemId":      "item-file",
				"startedAtMs": 123,
				"threadId":    "thread-1",
				"turnId":      "turn-1",
			},
			response: LegacyServerRequestResponse{ApprovalDecision: ApprovalAcceptForSession},
			assert: func(t *testing.T, req ServerRequest, result map[string]any) {
				if req.Kind != ServerRequestFileChangeApproval || req.FileChangeApproval == nil || req.ItemID != "item-file" {
					t.Fatalf("file change request = %#v", req)
				}
				if result["decision"] != "acceptForSession" {
					t.Fatalf("file change response = %#v", result)
				}
			},
		},
		{
			name:   "permissions typed response",
			method: protocolv2.MethodItemPermissionsRequestApproval,
			params: map[string]any{
				"cwd":         "/repo",
				"itemId":      "item-permissions",
				"permissions": map[string]any{"network": map[string]any{"enabled": true}},
				"startedAtMs": 123,
				"threadId":    "thread-1",
				"turnId":      "turn-1",
			},
			response: LegacyServerRequestResponse{
				PermissionsApproval: &protocolv2.PermissionsRequestApprovalResponse{
					Permissions: protocolv2.GrantedPermissionProfile{
						Network: protocolv2.Value(protocolv2.AdditionalNetworkPermissions{
							Enabled: protocolv2.Value(true),
						}),
					},
				},
			},
			assert: func(t *testing.T, req ServerRequest, result map[string]any) {
				if req.Kind != ServerRequestPermissionsApproval || req.PermissionsApproval == nil || req.Approval.CWD != "/repo" {
					t.Fatalf("permissions request = %#v", req)
				}
				if !reflect.DeepEqual(req.Approval.AvailableDecisions, []ApprovalDecision{ApprovalDecline}) {
					t.Fatalf("permissions available decisions = %#v", req.Approval.AvailableDecisions)
				}
				permissions := result["permissions"].(map[string]any)
				network := permissions["network"].(map[string]any)
				if network["enabled"] != true {
					t.Fatalf("permissions response = %#v", result)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			writer := &recordingWriteCloser{}
			var seen ServerRequest
			c := &client{
				stdin: writer,
				options: ClientOptions{LegacyServerRequestHandler: func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
					seen = req
					return tc.response, nil
				}},
			}
			c.respondToServerRequest("server-1", tc.method, tc.params)
			var response map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(writer.Bytes()), &response); err != nil {
				t.Fatalf("typed approval response JSON = %q: %v", writer.String(), err)
			}
			result, ok := response["result"].(map[string]any)
			if !ok {
				t.Fatalf("response missing result object: %#v", response)
			}
			tc.assert(t, seen, result)
		})
	}
}

func TestServerRequestHandlerTypedNonApprovalResponses(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		params   map[string]any
		response LegacyServerRequestResponse
		assert   func(t *testing.T, req ServerRequest, result map[string]any)
	}{
		{
			name:   "chatgpt auth refresh",
			method: protocolv2.MethodAccountChatGPTAuthTokensRefresh,
			params: fakeAuthRefreshParams(),
			response: LegacyServerRequestResponse{
				ChatGPTAuthTokensRefresh: &protocolv2.ChatgptAuthTokensRefreshResponse{
					AccessToken:      "access-token",
					ChatGPTAccountID: "account-1",
					ChatGPTPlanType:  protocolv2.Value("team"),
				},
			},
			assert: func(t *testing.T, req ServerRequest, result map[string]any) {
				if req.Kind != ServerRequestChatGPTAuthRefresh || req.ChatGPTAuthTokensRefresh == nil ||
					req.ChatGPTAuthTokensRefresh.Reason != protocolv2.ChatgptAuthTokensRefreshReasonUnauthorized {
					t.Fatalf("auth refresh request = %#v", req)
				}
				if result["accessToken"] != "access-token" || result["chatgptAccountId"] != "account-1" || result["chatgptPlanType"] != "team" {
					t.Fatalf("auth refresh response = %#v", result)
				}
			},
		},
		{
			name:   "tool call",
			method: protocolv2.MethodItemToolCall,
			params: fakeDynamicToolCallParams("thread-1", "turn-tool"),
			response: LegacyServerRequestResponse{
				DynamicToolCall: &protocolv2.DynamicToolCallResponse{
					ContentItems: []protocolv2.DynamicToolCallOutputContentItem{
						protocolv2.NewDynamicToolCallOutputContentItemInputText(protocolv2.DynamicToolCallOutputContentItemInputText{Text: "ok"}),
					},
					Success: true,
				},
			},
			assert: func(t *testing.T, req ServerRequest, result map[string]any) {
				if req.Kind != ServerRequestToolCall || req.DynamicToolCall == nil || req.ThreadID != "thread-1" ||
					req.TurnID != "turn-tool" || req.ItemID != "call-tool" || req.DynamicToolCall.Tool != "lookup" {
					t.Fatalf("tool call request = %#v", req)
				}
				contentItems, _ := result["contentItems"].([]any)
				first, _ := contentItems[0].(map[string]any)
				if result["success"] != true || first["type"] != "inputText" || first["text"] != "ok" {
					t.Fatalf("tool call response = %#v", result)
				}
			},
		},
		{
			name:   "tool user input",
			method: protocolv2.MethodItemToolRequestUserInput,
			params: fakeToolUserInputParams("thread-1", "turn-input"),
			response: LegacyServerRequestResponse{
				ToolRequestUserInput: &protocolv2.ToolRequestUserInputResponse{
					Answers: map[string]protocolv2.ToolRequestUserInputAnswer{
						"q1": {Answers: []string{"yes"}},
					},
				},
			},
			assert: func(t *testing.T, req ServerRequest, result map[string]any) {
				if req.Kind != ServerRequestUserInput || req.ToolRequestUserInput == nil || req.ThreadID != "thread-1" ||
					req.TurnID != "turn-input" || req.ItemID != "item-input" || len(req.ToolRequestUserInput.Questions) != 1 {
					t.Fatalf("user input request = %#v", req)
				}
				answers := result["answers"].(map[string]any)
				q1 := answers["q1"].(map[string]any)
				values := q1["answers"].([]any)
				if values[0] != "yes" {
					t.Fatalf("user input response = %#v", result)
				}
			},
		},
		{
			name:   "mcp elicitation",
			method: protocolv2.MethodMCPServerElicitationRequest,
			params: fakeMCPElicitationParams("thread-1", "turn-mcp"),
			response: LegacyServerRequestResponse{
				MCPElicitation: &protocolv2.McpServerElicitationRequestResponse{
					Action:  protocolv2.McpServerElicitationActionAccept,
					Content: jsonValuePtr(protocolv2.JSONObject(map[string]protocolv2.JSONValue{"name": protocolv2.JSONString("Ada")})),
				},
			},
			assert: func(t *testing.T, req ServerRequest, result map[string]any) {
				if req.Kind != ServerRequestMCPElicitation || req.MCPElicitation == nil || req.ThreadID != "thread-1" ||
					req.TurnID != "turn-mcp" || req.MCPElicitation.ServerName != "mcp-server" {
					t.Fatalf("mcp elicitation request = %#v", req)
				}
				content := result["content"].(map[string]any)
				if result["action"] != "accept" || content["name"] != "Ada" {
					t.Fatalf("mcp elicitation response = %#v", result)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			writer := &recordingWriteCloser{}
			var seen ServerRequest
			c := &client{
				stdin: writer,
				options: ClientOptions{LegacyServerRequestHandler: func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
					seen = req
					return tc.response, nil
				}},
			}
			c.respondToServerRequest("server-1", tc.method, tc.params)
			var response map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(writer.Bytes()), &response); err != nil {
				t.Fatalf("typed non-approval response JSON = %q: %v", writer.String(), err)
			}
			if errorObject, ok := response["error"]; ok {
				t.Fatalf("typed non-approval returned error: %#v", errorObject)
			}
			result, ok := response["result"].(map[string]any)
			if !ok {
				t.Fatalf("response missing result object: %#v", response)
			}
			tc.assert(t, seen, result)
		})
	}
}

func TestServerRequestPermissionsCancelShortcutFailsClosed(t *testing.T) {
	writer := &recordingWriteCloser{}
	c := &client{
		stdin:         writer,
		streams:       map[string]map[*threadStreamState]struct{}{},
		pendingErrors: map[string]error{},
		options: ClientOptions{LegacyServerRequestHandler: func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
			return LegacyServerRequestResponse{ApprovalDecision: ApprovalCancel}, nil
		}},
	}

	c.respondToServerRequest("server-1", protocolv2.MethodItemPermissionsRequestApproval, map[string]any{
		"cwd":         "/repo",
		"itemId":      "item-permissions",
		"permissions": map[string]any{"network": map[string]any{"enabled": true}},
		"startedAtMs": 123,
		"threadId":    "thread-1",
		"turnId":      "turn-1",
	})

	var response map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(writer.Bytes()), &response); err != nil {
		t.Fatalf("permissions cancel response JSON = %q: %v", writer.String(), err)
	}
	result, _ := response["result"].(map[string]any)
	if _, ok := result["decision"]; ok {
		t.Fatalf("permissions cancel shortcut used decision field: %#v", result)
	}
	permissions, ok := result["permissions"].(map[string]any)
	if !ok || len(permissions) != 0 {
		t.Fatalf("permissions cancel fail-closed response = %#v", response)
	}
	if err := c.pendingErrors["turn-1"]; err == nil || !strings.Contains(err.Error(), "invalid approval decision") {
		t.Fatalf("permissions cancel route error = %#v", err)
	}
}

func TestServerRequestHandlerInvalidApprovalDecisionFailsClosed(t *testing.T) {
	writer := &recordingWriteCloser{}
	c := &client{
		stdin:         writer,
		streams:       map[string]map[*threadStreamState]struct{}{},
		pendingErrors: map[string]error{},
		options: ClientOptions{LegacyServerRequestHandler: func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
			return LegacyServerRequestResponse{ApprovalDecision: ApprovalDecision("bogus")}, nil
		}},
	}

	c.respondToServerRequest("server-1", protocolv2.MethodItemCommandExecutionRequestApproval, fakeCommandApprovalParams("thread-1", "turn-1"))

	var response map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(writer.Bytes()), &response); err != nil {
		t.Fatalf("invalid approval decision response JSON = %q: %v", writer.String(), err)
	}
	result, _ := response["result"].(map[string]any)
	if response["id"] != "server-1" || result["decision"] != "decline" {
		t.Fatalf("invalid approval decision fail-closed response = %#v", response)
	}
}

func TestTypedServerRequestsRejectMalformedBeforeHandler(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		params  map[string]any
		message string
	}{
		{
			name:    "chatgpt auth refresh invalid enum",
			method:  protocolv2.MethodAccountChatGPTAuthTokensRefresh,
			params:  map[string]any{"reason": "expired"},
			message: "ChatgptAuthTokensRefreshReason",
		},
		{
			name:    "tool call missing arguments",
			method:  protocolv2.MethodItemToolCall,
			params:  map[string]any{"callId": "call-1", "threadId": "thread-1", "tool": "tool", "turnId": "turn-tool"},
			message: "DynamicToolCallParams.arguments",
		},
		{
			name:   "tool user input malformed question",
			method: protocolv2.MethodItemToolRequestUserInput,
			params: map[string]any{
				"itemId": "item-input",
				"questions": []any{
					map[string]any{"header": "Confirm", "id": "q1"},
				},
				"threadId": "thread-1",
				"turnId":   "turn-input",
			},
			message: "ToolRequestUserInputQuestion.question",
		},
		{
			name:    "mcp elicitation missing server name",
			method:  protocolv2.MethodMCPServerElicitationRequest,
			params:  map[string]any{"threadId": "thread-1", "turnId": "turn-mcp"},
			message: "McpServerElicitationRequestParams.serverName",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handlerCalled := make(chan struct{}, 1)
			writer := &recordingWriteCloser{}
			c := &client{
				stdin:         writer,
				streams:       map[string]map[*threadStreamState]struct{}{},
				pendingErrors: map[string]error{},
				options: ClientOptions{LegacyServerRequestHandler: func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
					handlerCalled <- struct{}{}
					return LegacyServerRequestResponse{}, errors.New("handler should not be called")
				}},
			}

			c.respondToServerRequest("server-1", tc.method, tc.params)

			select {
			case <-handlerCalled:
				t.Fatal("malformed server request invoked handler")
			default:
			}
			var response map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(writer.Bytes()), &response); err != nil {
				t.Fatalf("malformed response JSON = %q: %v", writer.String(), err)
			}
			errorObject, _ := response["error"].(map[string]any)
			if response["id"] != "server-1" || errorObject["code"] != float64(-32602) ||
				!strings.Contains(fmt.Sprint(errorObject["message"]), tc.message) {
				t.Fatalf("malformed error response = %#v", response)
			}
		})
	}
}

func TestTurnErrorsAndLocalCancellationAreDistinct(t *testing.T) {
	failed := newFakeClient(t, "failed", nil)
	_, err := failed.StartThread(context.Background(), StartThreadRequest{Input: Text("fail")})
	if err == nil ||
		!strings.Contains(err.Error(), "codexsdk: turn failed") ||
		!strings.Contains(err.Error(), "thread_id=thread-1") ||
		!strings.Contains(err.Error(), "turn_id=turn-1") ||
		!strings.Contains(err.Error(), "status=failed") ||
		!strings.Contains(err.Error(), "code=usageLimitExceeded") ||
		!strings.Contains(err.Error(), `message="native failed"`) {
		t.Fatalf("failed turn err = %v", err)
	}
	_ = failed.Close()

	interrupted := newFakeClient(t, "interrupted", nil)
	_, err = interrupted.StartThread(context.Background(), StartThreadRequest{Input: Text("interrupt")})
	if err == nil ||
		!strings.Contains(err.Error(), "codexsdk: turn interrupted") ||
		!strings.Contains(err.Error(), "thread_id=thread-1") ||
		!strings.Contains(err.Error(), "turn_id=turn-1") ||
		!strings.Contains(err.Error(), "status=interrupted") {
		t.Fatalf("interrupted turn err = %v", err)
	}
	_ = interrupted.Close()

	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("hang")})
	if err != nil {
		t.Fatal(err)
	}
	hanging := root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"})
	stream, err := hanging.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("hang")})
	if err != nil {
		t.Fatal(err)
	}
	if !stream.Next(context.Background()) || stream.Event().Kind != ThreadEventStarted {
		t.Fatal("missing started event")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if stream.Next(ctx) {
		t.Fatal("Next unexpectedly returned an event after started event was consumed")
	}
	if !errors.Is(stream.Err(), context.DeadlineExceeded) {
		t.Fatalf("stream err = %v, want deadline", stream.Err())
	}
	if !waitForRecord(t, record, "recv", "turn/interrupt", time.Second) {
		t.Fatalf("context deadline did not best-effort interrupt turn; records=%#v", readRecords(t, record))
	}
	_ = root.Close()
}

func TestTransportErrorDoesNotExposeStderrTail(t *testing.T) {
	client := newFakeClient(t, "stderr-exit", nil)
	defer client.Close()

	_, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("close")})
	if err == nil {
		t.Fatal("StartThread succeeded; want transport error")
	}
	for _, forbidden := range []string{"stderr_tail", "stderr diagnostic secret", "transcript"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("transport error leaked diagnostic stderr content %q: %v", forbidden, err)
		}
	}
}

func TestMalformedStdoutErrorDoesNotExposeRawLine(t *testing.T) {
	client := newFakeClient(t, "malformed-stdout", nil)
	defer client.Close()

	_, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("malformed")})
	if err == nil {
		t.Fatal("StartThread succeeded; want malformed stdout error")
	}
	for _, forbidden := range []string{"stdout_secret", "transcript", "{not-json"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Fatalf("malformed stdout error leaked raw content %q: %v", forbidden, err)
		}
	}
	if !strings.Contains(err.Error(), "invalid app-server JSON-RPC line bytes=") || !strings.Contains(err.Error(), "sha256=") {
		t.Fatalf("malformed stdout error missing sanitized metadata: %v", err)
	}
}

func TestServerRequestFailClosedApprovalHandlerAndUnsupported(t *testing.T) {
	record := tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err := New(ClientOptions{CWD: t.TempDir(), Command: fakeCommand("approval")})
	if err != nil {
		t.Fatal(err)
	}
	client := root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"})
	if _, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("approval")}); err != nil {
		t.Fatalf("nil handler approval should be denied safely without SDK error: %v", err)
	}
	_ = root.Close()
	approvalResponse := firstRecord(readRecords(t, record), "recv-response", "")
	if decision := approvalResponse["result"].(map[string]any)["decision"]; decision != "decline" {
		t.Fatalf("nil handler decision = %#v", approvalResponse)
	}

	record = tempRecord(t)
	t.Setenv("CODEXSDK_FAKE_RECORD", record)
	root, err = New(ClientOptions{
		CWD:     t.TempDir(),
		Command: fakeCommand("approval"),
		LegacyServerRequestHandler: func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
			if req.Kind != ServerRequestCommandApproval || req.Approval == nil || req.CommandExecutionApproval == nil ||
				req.CommandExecutionApproval.Command == nil || req.CommandExecutionApproval.Command.Value == nil ||
				*req.CommandExecutionApproval.Command.Value != "echo ok" ||
				!reflect.DeepEqual(req.Approval.Command, []string{"echo ok"}) {
				return LegacyServerRequestResponse{}, fmt.Errorf("approval request = %#v", req)
			}
			return LegacyServerRequestResponse{ApprovalDecision: ApprovalAcceptForSession}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	client = root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"})
	if _, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("approval")}); err != nil {
		t.Fatal(err)
	}
	_ = root.Close()
	approvalResponse = firstRecord(readRecords(t, record), "recv-response", "")
	if decision := approvalResponse["result"].(map[string]any)["decision"]; decision != "acceptForSession" {
		t.Fatalf("handler decision = %#v", approvalResponse)
	}

	unsupported := newFakeClient(t, "unsupported", nil)
	_, err = unsupported.StartThread(context.Background(), StartThreadRequest{Input: Text("unsupported")})
	if err == nil ||
		!strings.Contains(err.Error(), "codexsdk: unsupported server request") ||
		!strings.Contains(err.Error(), "kind=unsupported") ||
		!strings.Contains(err.Error(), "turn_id=turn-1") {
		t.Fatalf("unsupported err = %v", err)
	}
	_ = unsupported.Close()

	handlerCalled := make(chan struct{}, 1)
	unsupported = newFakeClient(t, "unsupported", func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
		handlerCalled <- struct{}{}
		return LegacyServerRequestResponse{}, errors.New("handler observation failed")
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = unsupported.StartThread(ctx, StartThreadRequest{Input: Text("unsupported")})
	if err == nil ||
		!strings.Contains(err.Error(), "codexsdk: unsupported server request") ||
		!strings.Contains(err.Error(), "kind=unsupported") ||
		!strings.Contains(err.Error(), "turn_id=turn-1") {
		t.Fatalf("unsupported with handler error = %v", err)
	}
	if strings.Contains(err.Error(), "handler observation failed") {
		t.Fatalf("unsupported canonical error was overwritten by handler error: %v", err)
	}
	select {
	case <-handlerCalled:
		t.Fatal("non-approval unsupported request invoked handler")
	case <-time.After(50 * time.Millisecond):
	}
	_ = unsupported.Close()

	noTurnHandlerCalled := make(chan struct{}, 1)
	noTurn := newFakeClient(t, "unsupported-no-turn", func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
		noTurnHandlerCalled <- struct{}{}
		return LegacyServerRequestResponse{}, nil
	})
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = noTurn.StartThread(ctx, StartThreadRequest{Input: Text("unsupported no turn")})
	if err == nil ||
		!strings.Contains(err.Error(), "codexsdk: unsupported server request") ||
		!strings.Contains(err.Error(), "kind=unsupported") ||
		strings.Contains(err.Error(), "turn_id=") {
		t.Fatalf("no-turn unsupported err = %v", err)
	}
	select {
	case <-noTurnHandlerCalled:
		t.Fatal("no-turn unsupported request invoked handler")
	default:
	}
	_ = noTurn.Close()
}

func TestServerRequestHandlerDoesNotBlockReaderLoop(t *testing.T) {
	handlerSeen := make(chan struct{})
	releaseHandler := make(chan struct{})
	client := newFakeClient(t, "blocking-approval-concurrent", func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
		close(handlerSeen)
		select {
		case <-releaseHandler:
			return LegacyServerRequestResponse{ApprovalDecision: ApprovalDecline}, nil
		case <-ctx.Done():
			return LegacyServerRequestResponse{}, ctx.Err()
		}
	})
	defer client.Close()

	stream, err := client.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("blocked approval")})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if !stream.Next(context.Background()) || stream.Event().Kind != ThreadEventStarted {
		t.Fatal("missing started event")
	}
	select {
	case <-handlerSeen:
	case <-time.After(time.Second):
		t.Fatal("server request handler was not invoked")
	}

	resultCh := make(chan error, 1)
	go func() {
		result, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("unrelated")})
		if err != nil {
			resultCh <- err
			return
		}
		if result.FinalResponse != "final-turn-2" {
			resultCh <- fmt.Errorf("unrelated result = %#v", result)
			return
		}
		resultCh <- nil
	}()
	select {
	case err := <-resultCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("unrelated turn was blocked by server request handler")
	}
	close(releaseHandler)
}

func TestTypedNonApprovalServerRequestBeforeAttachUsesStreamContextOnClose(t *testing.T) {
	handlerSeen := make(chan struct{})
	handlerDone := make(chan error, 1)
	client := newFakeClient(t, "tool-call-before-attach", func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
		if req.Kind != ServerRequestToolCall || req.DynamicToolCall == nil || req.TurnID == "" {
			handlerDone <- fmt.Errorf("typed tool server request = %#v", req)
			return LegacyServerRequestResponse{}, nil
		}
		close(handlerSeen)
		<-ctx.Done()
		handlerDone <- ctx.Err()
		return LegacyServerRequestResponse{}, ctx.Err()
	})
	defer client.Close()

	stream, err := client.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("tool call before attach")})
	if err != nil {
		t.Fatal(err)
	}
	if !stream.Next(context.Background()) || stream.Event().Kind != ThreadEventStarted {
		t.Fatal("missing started event")
	}
	select {
	case <-handlerSeen:
	case err := <-handlerDone:
		t.Fatalf("handler ended before stream close: %v", err)
	case <-time.After(time.Second):
		t.Fatal("typed non-approval server request handler was not invoked")
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-handlerDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("handler context err = %v, want canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("typed non-approval handler context was not canceled")
	}
	if !errors.Is(stream.Err(), ErrStreamClosed) {
		t.Fatalf("stream err = %v, want ErrStreamClosed", stream.Err())
	}
}

func TestServerRequestHandlerContextCancelsOnClientClose(t *testing.T) {
	handlerSeen := make(chan struct{})
	handlerCanceled := make(chan struct{})
	releaseHandler := make(chan struct{})
	handlerDone := make(chan error, 1)
	client := newFakeClient(t, "blocking-approval-concurrent", func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
		close(handlerSeen)
		<-ctx.Done()
		close(handlerCanceled)
		<-releaseHandler
		handlerDone <- ctx.Err()
		return LegacyServerRequestResponse{}, ctx.Err()
	})

	stream, err := client.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("blocked approval")})
	if err != nil {
		t.Fatal(err)
	}
	if !stream.Next(context.Background()) || stream.Event().Kind != ThreadEventStarted {
		t.Fatal("missing started event")
	}
	select {
	case <-handlerSeen:
	case <-time.After(time.Second):
		t.Fatal("server request handler was not invoked")
	}
	closed := make(chan error, 1)
	go func() { closed <- client.Close() }()
	<-handlerCanceled
	select {
	case err := <-closed:
		t.Fatalf("Close returned before admitted legacy handler completed: %v", err)
	default:
	}
	close(releaseHandler)
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-handlerDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("handler context err = %v, want canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server request handler context was not canceled")
	}
}

func TestServerRequestHandlerContextCancelsOnStreamClose(t *testing.T) {
	handlerSeen := make(chan struct{})
	handlerDone := make(chan error, 1)
	client := newFakeClient(t, "blocking-approval-delayed", func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
		close(handlerSeen)
		<-ctx.Done()
		handlerDone <- ctx.Err()
		return LegacyServerRequestResponse{}, ctx.Err()
	})
	defer client.Close()

	stream, err := client.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("blocked approval")})
	if err != nil {
		t.Fatal(err)
	}
	if !stream.Next(context.Background()) || stream.Event().Kind != ThreadEventStarted {
		t.Fatal("missing started event")
	}
	select {
	case <-handlerSeen:
	case <-time.After(time.Second):
		t.Fatal("server request handler was not invoked")
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-handlerDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("handler context err = %v, want canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server request handler context was not canceled")
	}
	if !errors.Is(stream.Err(), ErrStreamClosed) {
		t.Fatalf("stream err = %v, want ErrStreamClosed", stream.Err())
	}
}

func TestPendingServerRequestBeforeAttachUsesStreamContextOnClose(t *testing.T) {
	handlerSeen := make(chan struct{})
	handlerDone := make(chan error, 1)
	client := newFakeClient(t, "approval-before-attach", func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
		close(handlerSeen)
		<-ctx.Done()
		handlerDone <- ctx.Err()
		return LegacyServerRequestResponse{}, ctx.Err()
	})
	defer client.Close()

	stream, err := client.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("blocked approval")})
	if err != nil {
		t.Fatal(err)
	}
	if !stream.Next(context.Background()) || stream.Event().Kind != ThreadEventStarted {
		t.Fatal("missing started event")
	}
	select {
	case <-handlerSeen:
	case <-time.After(time.Second):
		t.Fatal("server request handler was not invoked")
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-handlerDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("handler context err = %v, want canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server request handler context was not canceled")
	}
	if !errors.Is(stream.Err(), ErrStreamClosed) {
		t.Fatalf("stream err = %v, want ErrStreamClosed", stream.Err())
	}
}

func TestServerRequestHandlerContextCancelsOnNextTimeout(t *testing.T) {
	handlerSeen := make(chan struct{})
	handlerDone := make(chan error, 1)
	client := newFakeClient(t, "blocking-approval-delayed", func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
		close(handlerSeen)
		<-ctx.Done()
		handlerDone <- ctx.Err()
		return LegacyServerRequestResponse{}, ctx.Err()
	})
	defer client.Close()

	stream, err := client.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("blocked approval")})
	if err != nil {
		t.Fatal(err)
	}
	if !stream.Next(context.Background()) || stream.Event().Kind != ThreadEventStarted {
		t.Fatal("missing started event")
	}
	select {
	case <-handlerSeen:
	case <-time.After(time.Second):
		t.Fatal("server request handler was not invoked")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if stream.Next(ctx) {
		t.Fatal("Next unexpectedly returned an event")
	}
	select {
	case err := <-handlerDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("handler context err = %v, want canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server request handler context was not canceled")
	}
	if !errors.Is(stream.Err(), context.DeadlineExceeded) {
		t.Fatalf("stream err = %v, want deadline", stream.Err())
	}
}

func TestPendingServerRequestBeforeAttachUsesStreamContextOnNextTimeout(t *testing.T) {
	handlerSeen := make(chan struct{})
	handlerDone := make(chan error, 1)
	client := newFakeClient(t, "approval-before-attach", func(ctx context.Context, req ServerRequest) (LegacyServerRequestResponse, error) {
		close(handlerSeen)
		<-ctx.Done()
		handlerDone <- ctx.Err()
		return LegacyServerRequestResponse{}, ctx.Err()
	})
	defer client.Close()

	stream, err := client.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("blocked approval")})
	if err != nil {
		t.Fatal(err)
	}
	if !stream.Next(context.Background()) || stream.Event().Kind != ThreadEventStarted {
		t.Fatal("missing started event")
	}
	select {
	case <-handlerSeen:
	case <-time.After(time.Second):
		t.Fatal("server request handler was not invoked")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if stream.Next(ctx) {
		t.Fatal("Next unexpectedly returned an event")
	}
	select {
	case err := <-handlerDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("handler context err = %v, want canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("server request handler context was not canceled")
	}
	if !errors.Is(stream.Err(), context.DeadlineExceeded) {
		t.Fatalf("stream err = %v, want deadline", stream.Err())
	}
}

func TestCloseIdempotencyAndConcurrentRouting(t *testing.T) {
	closed := newFakeClient(t, "hang", nil)
	stream, err := closed.StartThreadStream(context.Background(), StartThreadRequest{Input: Text("hang")})
	if err != nil {
		t.Fatal(err)
	}
	if !stream.Next(context.Background()) || stream.Event().Kind != ThreadEventStarted {
		t.Fatal("missing started event")
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(stream.Err(), ErrStreamClosed) {
		t.Fatalf("stream err = %v, want ErrStreamClosed", stream.Err())
	}
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := closed.ForkThread(context.Background(), ForkThreadRequest{ParentThreadID: "thread"}); !errors.Is(err, ErrClientClosed) {
		t.Fatalf("post-close err = %v, want ErrClientClosed", err)
	}

	client := newFakeClient(t, "concurrent", nil)
	defer client.Close()
	var wg sync.WaitGroup
	results := make(chan string, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := client.StartThread(context.Background(), StartThreadRequest{Input: Text("concurrent")})
			if err != nil {
				t.Errorf("StartThread error: %v", err)
				return
			}
			results <- result.FinalResponse
		}()
	}
	wg.Wait()
	close(results)
	got := map[string]bool{}
	for result := range results {
		got[result] = true
	}
	if !got["final-turn-1"] || !got["final-turn-2"] {
		t.Fatalf("concurrent routed results = %#v", got)
	}
}

type testThreadClient struct {
	ThreadClient
	close func() error
}

func (c testThreadClient) Close() error {
	return c.close()
}

func newFakeClient(t *testing.T, mode string, handler LegacyServerRequestHandler) testThreadClient {
	t.Helper()
	t.Setenv("CODEXSDK_FAKE_RECORD", tempRecord(t))
	root, err := New(ClientOptions{
		CWD:                        t.TempDir(),
		Command:                    fakeCommand(mode),
		LegacyServerRequestHandler: handler,
	})
	if err != nil {
		t.Fatalf("New(%s) error: %v", mode, err)
	}
	return testThreadClient{
		ThreadClient: root.ThreadClient(ThreadClientOptions{DefaultModel: "client-model"}),
		close:        root.Close,
	}
}

func fakeCommand(mode string, extra ...string) []string {
	args := []string{os.Args[0], "-test.run=TestHelperProcess", "--", mode}
	args = append(args, extra...)
	return args
}

func tempRecord(t *testing.T) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "codexsdk-record-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func readRecords(t *testing.T, path string) []map[string]any {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var records []map[string]any
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			t.Fatalf("bad record %q: %v", scanner.Text(), err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return records
}

func firstRecord(records []map[string]any, kind, method string) map[string]any {
	for _, record := range records {
		if record["kind"] != kind {
			continue
		}
		if method == "" || record["method"] == method {
			return record
		}
	}
	return nil
}

func recordsByMethod(records []map[string]any, kind, method string) []map[string]any {
	var out []map[string]any
	for _, record := range records {
		if record["kind"] == kind && record["method"] == method {
			out = append(out, record)
		}
	}
	return out
}

func mustOutputSchema(t *testing.T, raw string) protocolv2.OutputSchema {
	t.Helper()
	schema, err := protocolv2.OutputSchemaFromJSON([]byte(raw))
	if err != nil {
		t.Fatalf("parse output schema %s: %v", raw, err)
	}
	return schema
}

func assertJSONEqual(t *testing.T, got any, want string) {
	t.Helper()
	gotRaw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal got JSON value: %v", err)
	}
	var gotValue any
	if err := json.Unmarshal(gotRaw, &gotValue); err != nil {
		t.Fatalf("unmarshal got JSON value %q: %v", gotRaw, err)
	}
	var wantValue any
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("unmarshal want JSON value %q: %v", want, err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("JSON value = %#v, want %#v", gotValue, wantValue)
	}
}

func waitForRecord(t *testing.T, path, kind, method string, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if firstRecord(readRecords(t, path), kind, method) != nil {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

func stringify(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.(string))
	}
	return out
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("CODEXSDK_FAKE_RECORD") == "" {
		return
	}
	idx := 0
	for i, arg := range os.Args {
		if arg == "--" {
			idx = i + 1
			break
		}
	}
	if idx == 0 || idx >= len(os.Args) {
		return
	}
	runFakeAppServer(os.Args[idx], os.Args[idx+1:])
	os.Exit(0)
}

func runFakeAppServer(mode string, extra []string) {
	record := os.Getenv("CODEXSDK_FAKE_RECORD")
	appendRecord(record, map[string]any{"kind": "process", "argv": os.Args[1:], "cwd": mustGetwd(), "extra": extra})
	scanner := bufio.NewScanner(os.Stdin)
	threadCounter := 0
	turnCounter := 0
	pendingConcurrent := []string{}
	for scanner.Scan() {
		var message map[string]any
		_ = json.Unmarshal(scanner.Bytes(), &message)
		method, _ := message["method"].(string)
		if method == "" {
			appendRecord(record, map[string]any{"kind": "recv-response", "result": message["result"], "error": message["error"]})
			if mode == "approval" || mode == "approval-before-turn-start" || mode == "file-approval" || mode == "user-input" || mode == "notification-and-approval" || mode == "late-approval-during-close" {
				completeTurn("thread-1", "turn-1")
			}
			continue
		}
		appendRecord(record, map[string]any{"kind": "recv", "method": method, "params": message["params"]})
		id := message["id"]
		switch method {
		case "account/login/cancel":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.CancelLoginAccountResponse{
					Status: protocolv2.CancelLoginAccountStatusCanceled,
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"status": "canceled"}})
		case "account/login/start":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.NewLoginAccountResponseChatGPT(protocolv2.LoginAccountResponseChatGPT{
					AuthURL: "https://example.test/auth",
					LoginID: "login-1",
				}))
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"authUrl": "https://example.test/auth", "loginId": "login-1", "type": "chatgpt"}})
		case "account/logout":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.LogoutAccountResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "account/rateLimits/read":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.GetAccountRateLimitsResponse{
					RateLimits: protocolv2.RateLimitSnapshot{
						PlanType: protocolv2.Value(protocolv2.PlanTypePlus),
					},
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"rateLimits": map[string]any{"planType": "plus"}}})
		case "account/rateLimitResetCredit/consume":
			send(map[string]any{"id": id, "result": map[string]any{"outcome": "reset"}})
		case "account/read":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.GetAccountResponse{
					Account:            protocolv2.Value(protocolv2.NewAccountAPIKey()),
					RequiresOpenaiAuth: false,
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"account": map[string]any{"email": "user@example.test", "planType": "plus", "type": "chatgpt"}, "requiresOpenaiAuth": false}})
		case "account/sendAddCreditsNudgeEmail":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.SendAddCreditsNudgeEmailResponse{
					Status: protocolv2.AddCreditsNudgeEmailStatusSent,
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"status": "sent"}})
		case "app/list":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.AppsListResponse{
					Data: []protocolv2.AppInfo{{ID: "app-1", Name: "App One"}},
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"data": []map[string]any{}}})
		case "collaborationMode/list":
			if mode == "collaboration-modes-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"nextCursor": "cursor-1"}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, facadeCollaborationModeListResponse())
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"data": []map[string]any{}}})
		case "command/exec":
			if mode == "command-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"exitCode": 0, "stdout": "ok\n"}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.CommandExecResponse{
					ExitCode: 0,
					Stderr:   "",
					Stdout:   "ok\n",
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"exitCode": 0, "stderr": "", "stdout": "ok\n"}})
		case "command/exec/resize":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.CommandExecResizeResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "command/exec/terminate":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.CommandExecTerminateResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "command/exec/write":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.CommandExecWriteResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "config/batchWrite":
			if mode == "facade" {
				sendProtocolResult(id, facadeConfigWriteResponse())
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"filePath": "/home/user/.codex/config.toml", "status": "ok", "version": "v2"}})
		case "config/mcpServer/reload":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.McpServerRefreshResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "config/read":
			if mode == "config-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"config": map[string]any{"model": "gpt-5"}}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, facadeConfigReadResponse())
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"config": map[string]any{"model": "gpt-5"}, "origins": map[string]any{}}})
		case "config/value/write":
			if mode == "facade" {
				sendProtocolResult(id, facadeConfigWriteResponse())
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"filePath": "/home/user/.codex/config.toml", "status": "ok", "version": "v2"}})
		case "configRequirements/read":
			if mode == "config-requirements-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, facadeConfigRequirementsReadResponse())
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"requirements": map[string]any{}}})
		case "experimentalFeature/enablement/set":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ExperimentalFeatureEnablementSetResponse{
					Enablement: map[string]bool{
						"feature_a": true,
						"feature_b": false,
					},
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"enablement": map[string]bool{"feature_a": true}}})
		case "experimentalFeature/list":
			if mode == "experimental-features-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"nextCursor": "cursor-2"}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, facadeExperimentalFeatureListResponse())
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"data": []map[string]any{}}})
		case "externalAgentConfig/detect":
			if mode == "external-agent-configs-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, facadeExternalAgentConfigDetectResponse())
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"items": []map[string]any{}}})
		case "externalAgentConfig/import":
			if mode == "facade" {
				send(map[string]any{"id": id, "result": map[string]any{}})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"importId": "import-1"}})
		case "feedback/upload":
			if mode == "feedback-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FeedbackUploadResponse{ThreadID: "thread-1"})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"threadId": "thread-1"}})
		case "fuzzyFileSearch":
			if mode == "fuzzy-file-search-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FuzzyFileSearchResponse{
					Files: []protocolv2.FuzzyFileSearchResult{{
						FileName:  "README.md",
						Indices:   protocolv2.Value([]uint32{0, 2}),
						MatchType: protocolv2.FuzzyFileSearchMatchTypeFile,
						Path:      "/repo/README.md",
						Root:      "/repo",
						Score:     99,
					}},
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"files": []map[string]any{}}})
		case "fuzzyFileSearch/sessionStart":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FuzzyFileSearchSessionStartResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "fuzzyFileSearch/sessionStop":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FuzzyFileSearchSessionStopResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "fuzzyFileSearch/sessionUpdate":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FuzzyFileSearchSessionUpdateResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "hooks/list":
			if mode == "hooks-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, facadeHooksListResponse())
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"data": []map[string]any{}}})
		case "marketplace/add":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.MarketplaceAddResponse{
					AlreadyAdded:    false,
					InstalledRoot:   "/codex/marketplaces/local",
					MarketplaceName: "local",
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"alreadyAdded": false, "installedRoot": "/codex/marketplaces/local", "marketplaceName": "local"}})
		case "marketplace/remove":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.MarketplaceRemoveResponse{
					InstalledRoot:   protocolv2.Value("/codex/marketplaces/local"),
					MarketplaceName: "local",
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"installedRoot": "/codex/marketplaces/local", "marketplaceName": "local"}})
		case "marketplace/upgrade":
			if mode == "marketplace-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"selectedMarketplaces": []string{"local"}, "upgradedRoots": []string{"/codex/marketplaces/local"}}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.MarketplaceUpgradeResponse{
					Errors:               []protocolv2.MarketplaceUpgradeErrorInfo{},
					SelectedMarketplaces: []string{"local"},
					UpgradedRoots:        []string{"/codex/marketplaces/local"},
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"errors": []map[string]any{}, "selectedMarketplaces": []string{"local"}, "upgradedRoots": []string{"/codex/marketplaces/local"}}})
		case "memory/reset":
			if mode == "memory-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.MemoryResetResponse{})
		case "mock/experimentalMethod":
			if mode == "mock-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			var echoed *protocolv2.Nullable[string]
			if params, ok := message["params"].(map[string]any); ok {
				if value, ok := params["value"]; ok {
					if value == nil {
						echoed = protocolv2.Null[string]()
					} else if text, ok := value.(string); ok {
						echoed = protocolv2.Value(text)
					}
				}
			}
			sendProtocolResult(id, protocolv2.MockExperimentalMethodResponse{Echoed: echoed})
		case "plugin/install":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.PluginInstallResponse{
					AppsNeedingAuth: []protocolv2.AppSummary{{
						ID:   "app-1",
						Name: "App One",
					}},
					AuthPolicy: protocolv2.PluginAuthPolicyONINSTALL,
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"appsNeedingAuth": []map[string]any{}, "authPolicy": "ON_INSTALL"}})
		case "plugin/list":
			if mode == "plugins-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"featuredPluginIds": []string{"plugin-1"}}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.PluginListResponse{
					FeaturedPluginIDs:     &[]string{"plugin-1"},
					MarketplaceLoadErrors: &[]protocolv2.MarketplaceLoadErrorInfo{},
					Marketplaces: []protocolv2.PluginMarketplaceEntry{{
						Name:    "market",
						Plugins: []protocolv2.PluginSummary{facadePluginSummary()},
					}},
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"marketplaces": []map[string]any{}}})
		case "plugin/read":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.PluginReadResponse{
					Plugin: facadePluginDetail(),
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"plugin": map[string]any{}}})
		case "plugin/share/delete":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.PluginShareDeleteResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "plugin/share/list":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.PluginShareListResponse{
					Data: []protocolv2.PluginShareListItem{{
						LocalPluginPath: protocolv2.Value("/plugins/plugin-one"),
						Plugin:          facadePluginSummary(),
					}},
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"data": []map[string]any{}}})
		case "plugin/share/save":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.PluginShareSaveResponse{
					RemotePluginID: "remote-1",
					ShareURL:       "https://example.test/plugin",
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"remotePluginId": "remote-1", "shareUrl": "https://example.test/plugin"}})
		case "plugin/share/updateTargets":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.PluginShareUpdateTargetsResponse{
					Discoverability: protocolv2.PluginShareDiscoverabilityPRIVATE,
					Principals: []protocolv2.PluginSharePrincipal{{
						Name:          "User One",
						PrincipalID:   "user-1",
						PrincipalType: protocolv2.PluginSharePrincipalTypeUser,
						Role:          protocolv2.PluginSharePrincipalRoleOwner,
					}},
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"discoverability": "PRIVATE", "principals": []map[string]any{}}})
		case "plugin/skill/read":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.PluginSkillReadResponse{
					Contents: protocolv2.Value("skill contents"),
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"contents": "skill contents"}})
		case "plugin/uninstall":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.PluginUninstallResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "process/kill":
			if mode == "process-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.ProcessKillResponse{})
		case "process/resizePty":
			if mode == "process-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.ProcessResizePtyResponse{})
		case "process/spawn":
			if mode == "process-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.ProcessSpawnResponse{})
		case "process/writeStdin":
			if mode == "process-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.ProcessWriteStdinResponse{})
		case "mcpServer/oauth/login":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.McpServerOauthLoginResponse{
					AuthorizationURL: "https://example.test/oauth",
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"authorizationUrl": "https://example.test/oauth"}})
		case "mcpServer/resource/read":
			if mode == "mcp-servers-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.McpResourceReadResponse{
					Contents: []protocolv2.ResourceContent{
						protocolv2.NewResourceContentText(protocolv2.ResourceContentText{
							Text: "hello",
							URI:  "file://README.md",
						}),
					},
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"contents": []map[string]any{{"text": "hello", "uri": "file://README.md"}}}})
		case "mcpServer/tool/call":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.McpServerToolCallResponse{
					Content: []protocolv2.JSONValue{protocolv2.JSONString("ok")},
					IsError: protocolv2.Value(false),
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"content": []string{"ok"}, "isError": false}})
		case "mcpServerStatus/list":
			if mode == "mcp-server-status-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"nextCursor": "cursor-2"}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ListMcpServerStatusResponse{
					Data: []protocolv2.McpServerStatus{{
						AuthStatus:        protocolv2.McpAuthStatusOAuth,
						Name:              "server-1",
						ResourceTemplates: []protocolv2.ResourceTemplate{},
						Resources:         []protocolv2.Resource{},
						Tools: map[string]protocolv2.Tool{
							"search": {
								InputSchema: protocolv2.JSONObject(map[string]protocolv2.JSONValue{"type": protocolv2.JSONString("object")}),
								Name:        "search",
							},
						},
					}},
					NextCursor: protocolv2.Value("cursor-2"),
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"data": []map[string]any{}, "nextCursor": "cursor-2"}})
		case "model/list":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ModelListResponse{
					Data:       []protocolv2.Model{},
					NextCursor: protocolv2.Value("model-cursor"),
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"data": []map[string]any{}}})
		case "modelProvider/capabilities/read":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ModelProviderCapabilitiesReadResponse{
					ImageGeneration: false,
					NamespaceTools:  true,
					WebSearch:       true,
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"imageGeneration": false, "namespaceTools": false, "webSearch": false}})
		case "review/start":
			if mode == "review-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"reviewThreadId": "review-thread-1"}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, facadeReviewStartResponse())
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"reviewThreadId": "review-thread-1", "turn": map[string]any{"id": "turn-review-1", "items": []map[string]any{}, "status": "inProgress"}}})
		case "skills/config/write":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.SkillsConfigWriteResponse{EffectiveEnabled: true})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"effectiveEnabled": false}})
		case "skills/list":
			if mode == "skills-list-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, facadeSkillsListResponse())
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"data": []map[string]any{}}})
		case "fs/copy":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FsCopyResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "fs/createDirectory":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FsCreateDirectoryResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "fs/getMetadata":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FsGetMetadataResponse{
					CreatedAtMS:  1000,
					IsDirectory:  false,
					IsFile:       true,
					IsSymlink:    false,
					ModifiedAtMS: 2000,
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "fs/readDirectory":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FsReadDirectoryResponse{
					Entries: []protocolv2.FsReadDirectoryEntry{{
						FileName:    "file.txt",
						IsDirectory: false,
						IsFile:      true,
					}},
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "fs/readFile":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FsReadFileResponse{DataBase64: "ZmFrZQ=="})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "fs/remove":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FsRemoveResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "fs/unwatch":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FsUnwatchResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "fs/watch":
			if mode == "facade" {
				params, _ := message["params"].(map[string]any)
				path, _ := params["path"].(string)
				sendProtocolResult(id, protocolv2.FsWatchResponse{Path: path})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "fs/writeFile":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.FsWriteFileResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "initialize":
			if mode == "initialize-error" {
				send(map[string]any{"id": id, "error": map[string]any{"code": -32000, "message": "initialize failed"}})
				continue
			}
			if mode == "initialize-close-stdout" {
				return
			}
			if mode == "initialize-malformed-stdout" {
				fmt.Fprintln(os.Stdout, "{not-json initialize_secret transcript")
				return
			}
			if mode == "bad-initialize" {
				send(map[string]any{"id": id, "result": map[string]any{"codexHome": "/tmp/codex-home", "platformFamily": "unix", "userAgent": "fake-app-server"}})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"codexHome": "/tmp/codex-home", "platformFamily": "unix", "platformOs": "darwin", "userAgent": "fake-app-server"}})
		case "initialized":
		case "thread/start":
			if mode == "stderr-exit" {
				fmt.Fprintln(os.Stderr, "stderr diagnostic secret transcript")
				return
			}
			if mode == "malformed-stdout" {
				fmt.Fprintln(os.Stdout, "{not-json stdout_secret transcript")
				return
			}
			threadCounter++
			if mode == "thread-start-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"thread": map[string]any{"id": "thread-" + itoa(threadCounter)}}})
				continue
			}
			if mode == "facade" {
				params, _ := message["params"].(map[string]any)
				model, _ := params["model"].(string)
				sendProtocolResult(id, facadeThreadStartResponse("thread-"+itoa(threadCounter), model))
				continue
			}
			params, _ := message["params"].(map[string]any)
			model, _ := params["model"].(string)
			sendProtocolResult(id, facadeThreadStartResponse("thread-"+itoa(threadCounter), model))
		case "thread/resume":
			params, _ := message["params"].(map[string]any)
			if mode == "facade" {
				threadID, _ := params["threadId"].(string)
				model, _ := params["model"].(string)
				sendProtocolResult(id, facadeThreadResumeResponse(threadID, model))
				continue
			}
			threadID, _ := params["threadId"].(string)
			model, _ := params["model"].(string)
			sendProtocolResult(id, facadeThreadResumeResponse(threadID, model))
		case "thread/fork":
			if mode == "facade" {
				params, _ := message["params"].(map[string]any)
				parentThreadID, _ := params["threadId"].(string)
				model, _ := params["model"].(string)
				sendProtocolResult(id, facadeThreadForkResponse("thread-fork", parentThreadID, model))
				continue
			}
			params, _ := message["params"].(map[string]any)
			parentThreadID, _ := params["threadId"].(string)
			model, _ := params["model"].(string)
			sendProtocolResult(id, facadeThreadForkResponse("thread-fork", parentThreadID, model))
		case "thread/goal/clear":
			if mode == "thread-goal-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadGoalClearResponse{Cleared: true})
		case "thread/goal/get":
			if mode == "thread-goal-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"goal": map[string]any{"threadId": "thread-1"}}})
				continue
			}
			params, _ := message["params"].(map[string]any)
			threadID, _ := params["threadId"].(string)
			sendProtocolResult(id, protocolv2.ThreadGoalGetResponse{
				Goal: protocolv2.Value(facadeThreadGoal(defaultString(threadID, "thread-1"), "ship protocol coverage", protocolv2.ThreadGoalStatusActive, protocolv2.Value(int64(4096)))),
			})
		case "thread/goal/set":
			if mode == "thread-goal-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"goal": map[string]any{"threadId": "thread-1"}}})
				continue
			}
			params, _ := message["params"].(map[string]any)
			threadID, _ := params["threadId"].(string)
			objective, _ := params["objective"].(string)
			statusText, _ := params["status"].(string)
			status := protocolv2.ThreadGoalStatus(statusText)
			if !status.IsValid() {
				status = protocolv2.ThreadGoalStatusActive
			}
			tokenBudget := protocolv2.Value(int64(4096))
			if value, ok := params["tokenBudget"].(float64); ok {
				tokenBudget = protocolv2.Value(int64(value))
			}
			sendProtocolResult(id, protocolv2.ThreadGoalSetResponse{
				Goal: facadeThreadGoal(defaultString(threadID, "thread-1"), defaultString(objective, "ship protocol coverage"), status, tokenBudget),
			})
		case "thread/increment_elicitation":
			if mode == "thread-support-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"paused": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadIncrementElicitationResponse{Count: 3, Paused: true})
		case "thread/decrement_elicitation":
			if mode == "thread-support-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"paused": false}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadDecrementElicitationResponse{Count: 2, Paused: false})
		case "thread/memoryMode/set":
			if mode == "thread-support-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadMemoryModeSetResponse{})
		case "thread/backgroundTerminals/clean":
			if mode == "thread-support-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadBackgroundTerminalsCleanResponse{})
		case "thread/turns/items/list":
			if mode == "thread-turns-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"nextCursor": "next-item"}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadTurnsItemsListResponse{
				BackwardsCursor: protocolv2.Null[string](),
				Data:            []protocolv2.ThreadItem{facadeThreadAgentMessageItem("item-agent-1", "final text")},
				NextCursor:      protocolv2.Value("next-item"),
			})
		case "thread/turns/list":
			if mode == "thread-turns-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"nextCursor": "next-turn"}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadTurnsListResponse{
				BackwardsCursor: protocolv2.Null[string](),
				Data:            []protocolv2.Turn{facadeThreadTurn("turn-list-1", []protocolv2.ThreadItem{facadeThreadAgentMessageItem("item-agent-1", "final text")})},
				NextCursor:      protocolv2.Value("next-turn"),
			})
		case "thread/realtime/start":
			if mode == "thread-realtime-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadRealtimeStartResponse{})
		case "thread/realtime/appendAudio":
			if mode == "thread-realtime-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadRealtimeAppendAudioResponse{})
		case "thread/realtime/appendText":
			if mode == "thread-realtime-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadRealtimeAppendTextResponse{})
		case "thread/realtime/listVoices":
			if mode == "thread-realtime-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadRealtimeListVoicesResponse{
				Voices: protocolv2.RealtimeVoicesList{
					DefaultV1: protocolv2.RealtimeVoiceAlloy,
					DefaultV2: protocolv2.RealtimeVoiceMarin,
					V1:        []protocolv2.RealtimeVoice{protocolv2.RealtimeVoiceAlloy},
					V2:        []protocolv2.RealtimeVoice{protocolv2.RealtimeVoiceMarin},
				},
			})
		case "thread/realtime/stop":
			if mode == "thread-realtime-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			sendProtocolResult(id, protocolv2.ThreadRealtimeStopResponse{})
		case "thread/approveGuardianDeniedAction":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ThreadApproveGuardianDeniedActionResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "thread/archive":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ThreadArchiveResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "thread/compact/start":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ThreadCompactStartResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "thread/inject_items":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ThreadInjectItemsResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "thread/list":
			if mode == "thread-list-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"nextCursor": "next-thread"}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ThreadListResponse{
					BackwardsCursor: protocolv2.Null[string](),
					Data:            []protocolv2.Thread{facadeThread("thread-list-1", nil)},
					NextCursor:      protocolv2.Value("next-thread"),
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"data": []any{}}})
		case "thread/loaded/list":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ThreadLoadedListResponse{
					Data:       []string{"thread-1"},
					NextCursor: protocolv2.Value("loaded-next"),
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"data": []any{}}})
		case "thread/metadata/update":
			if mode == "facade" {
				params, _ := message["params"].(map[string]any)
				threadID, _ := params["threadId"].(string)
				sendProtocolResult(id, protocolv2.ThreadMetadataUpdateResponse{Thread: facadeThread(defaultString(threadID, "thread-1"), nil)})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"thread": map[string]any{"id": "thread-1"}}})
		case "thread/name/set":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ThreadSetNameResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "thread/read":
			if mode == "facade" {
				params, _ := message["params"].(map[string]any)
				threadID, _ := params["threadId"].(string)
				sendProtocolResult(id, protocolv2.ThreadReadResponse{Thread: facadeThread(defaultString(threadID, "thread-1"), nil)})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"thread": map[string]any{"id": "thread-1"}}})
		case "thread/rollback":
			if mode == "facade" {
				params, _ := message["params"].(map[string]any)
				threadID, _ := params["threadId"].(string)
				sendProtocolResult(id, protocolv2.ThreadRollbackResponse{Thread: facadeThread(defaultString(threadID, "thread-1"), nil)})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"thread": map[string]any{"id": "thread-1"}}})
		case "thread/shellCommand":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ThreadShellCommandResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "thread/unarchive":
			if mode == "facade" {
				params, _ := message["params"].(map[string]any)
				threadID, _ := params["threadId"].(string)
				sendProtocolResult(id, protocolv2.ThreadUnarchiveResponse{Thread: facadeThread(defaultString(threadID, "thread-1"), nil)})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"thread": map[string]any{"id": "thread-1"}}})
		case "thread/unsubscribe":
			if mode == "thread-unsubscribe-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"status": "bogus"}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.ThreadUnsubscribeResponse{Status: protocolv2.ThreadUnsubscribeStatusUnsubscribed})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"status": "unsubscribed"}})
		case "turn/start":
			turnCounter++
			turnID := "turn-" + itoa(turnCounter)
			params, _ := message["params"].(map[string]any)
			threadID, _ := params["threadId"].(string)
			if mode == "turn-start-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"turn": map[string]any{"id": turnID, "status": "inProgress"}}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.TurnStartResponse{
					Turn: protocolv2.Turn{
						ID:     turnID,
						Items:  []protocolv2.ThreadItem{},
						Status: protocolv2.TurnStatusInProgress,
					},
				})
				continue
			}
			if mode == "approval-before-turn-start" {
				send(map[string]any{"id": "server-approval-1", "method": "item/commandExecution/requestApproval", "params": fakeCommandApprovalParams(threadID, turnID)})
			}
			if turnCounter == 2 && (mode == "exact-overflow-pending" || mode == "exact-overflow-terminal") {
				sendExactReroute(threadID, turnID, "model-a", "model-b")
				if mode == "exact-overflow-terminal" {
					send(map[string]any{"method": "turn/completed", "params": map[string]any{
						"threadId": threadID,
						"turn": map[string]any{
							"id": turnID, "status": "completed",
							"items": []map[string]any{{"id": "answer", "type": "agentMessage", "text": "done", "phase": "final_answer"}},
						},
					}})
				} else {
					sendExactReroute(threadID, turnID, "model-b", "model-c")
				}
			}
			sendProtocolResult(id, protocolv2.TurnStartResponse{
				Turn: protocolv2.Turn{
					ID:     turnID,
					Items:  []protocolv2.ThreadItem{},
					Status: protocolv2.TurnStatusInProgress,
				},
			})
			switch mode {
			case "exact-overflow-live":
				if turnCounter == 2 {
					for {
						if _, err := os.Stat(extra[0]); err == nil {
							break
						}
						time.Sleep(time.Millisecond)
					}
					sendExactReroute(threadID, turnID, "model-a", "model-b")
					sendExactReroute(threadID, turnID, "model-b", "model-c")
				}
			case "exact-overflow-pending", "exact-overflow-terminal":
			case "failed":
				send(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": threadID, "turn": map[string]any{"id": turnID, "items": []map[string]any{}, "status": "failed", "error": map[string]any{"message": "native failed", "codexErrorInfo": "usageLimitExceeded"}}}})
			case "interrupted":
				send(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": threadID, "turn": map[string]any{"id": turnID, "items": []map[string]any{}, "status": "interrupted"}}})
			case "delta-without-final":
				send(map[string]any{"method": "item/agentMessage/delta", "params": map[string]any{"threadId": threadID, "turnId": turnID, "itemId": "item-delta", "delta": "delta-only-output"}})
				send(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": threadID, "turn": map[string]any{"id": turnID, "status": "completed", "items": []map[string]any{}}}})
			case "turn-completed-in-progress":
				send(map[string]any{"method": "item/completed", "params": map[string]any{"completedAtMs": 1234, "threadId": threadID, "turnId": turnID, "item": map[string]any{"id": "item-" + turnID, "type": "agentMessage", "text": "non-terminal", "phase": "final_answer"}}})
				send(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": threadID, "turn": map[string]any{"id": turnID, "status": "inProgress", "items": []map[string]any{}}}})
			case "hang":
			case "approval":
				send(map[string]any{"id": "server-approval-1", "method": "item/commandExecution/requestApproval", "params": fakeCommandApprovalParams(threadID, turnID)})
			case "notification-and-approval":
				send(map[string]any{"method": "item/completed", "params": map[string]any{"completedAtMs": 1, "threadId": threadID, "turnId": turnID, "item": map[string]any{"id": "item-before-close", "type": "agentMessage", "text": "partial", "phase": "commentary"}}})
				send(map[string]any{"id": "server-approval-1", "method": "item/commandExecution/requestApproval", "params": fakeCommandApprovalParams(threadID, turnID)})
			case "late-approval-during-close":
				send(map[string]any{"id": "server-approval-1", "method": "item/commandExecution/requestApproval", "params": fakeCommandApprovalParams(threadID, turnID)})
				for {
					if _, err := os.Stat(extra[0]); err == nil {
						break
					}
					time.Sleep(time.Millisecond)
				}
				send(map[string]any{"id": "server-approval-late", "method": "item/commandExecution/requestApproval", "params": fakeCommandApprovalParams(threadID, turnID)})
			case "late-approval-during-failure":
				send(map[string]any{"method": "item/completed", "params": map[string]any{"completedAtMs": 1, "threadId": threadID, "turnId": turnID, "item": map[string]any{"id": "partial", "type": "agentMessage", "text": "partial", "phase": "commentary"}}})
				send(map[string]any{"id": "server-approval-1", "method": "item/commandExecution/requestApproval", "params": fakeCommandApprovalParams(threadID, turnID)})
				for {
					if _, err := os.Stat(extra[0]); err == nil {
						break
					}
					time.Sleep(time.Millisecond)
				}
				send(map[string]any{"id": "server-approval-late", "method": "item/commandExecution/requestApproval", "params": fakeCommandApprovalParams(threadID, turnID)})
				if err := os.WriteFile(extra[1], []byte("sent"), 0o600); err != nil {
					return
				}
			case "handler-error-then-transport-close":
				send(map[string]any{"method": "item/completed", "params": map[string]any{"completedAtMs": 1, "threadId": threadID, "turnId": turnID, "item": map[string]any{"id": "partial", "type": "agentMessage", "text": "partial", "phase": "commentary"}}})
				for {
					if _, err := os.Stat(extra[0]); err == nil {
						return
					}
					time.Sleep(time.Millisecond)
				}
			case "protocol-failure-multiple-streams":
				send(map[string]any{"method": "item/completed", "params": map[string]any{"completedAtMs": int64(turnCounter), "threadId": threadID, "turnId": turnID, "item": map[string]any{"id": "partial-" + turnID, "type": "agentMessage", "text": "partial", "phase": "commentary"}}})
				if turnCounter == 2 {
					for {
						if _, err := os.Stat(extra[0]); err == nil {
							break
						}
						time.Sleep(time.Millisecond)
					}
					_, _ = fmt.Fprintln(os.Stdout, "{")
					return
				}
			case "approval-before-turn-start":
			case "file-approval":
				send(map[string]any{"id": "server-file-1", "method": "item/fileChange/requestApproval", "params": map[string]any{"itemId": "item-file", "startedAtMs": 1, "threadId": threadID, "turnId": turnID}})
			case "user-input":
				send(map[string]any{"id": "server-input-1", "method": "item/tool/requestUserInput", "params": map[string]any{"itemId": "item-input", "questions": []map[string]any{{"header": "Choice", "id": "choice", "question": "Choose"}}, "threadId": threadID, "turnId": turnID}})
			case "auth-refresh-after-notification":
				send(map[string]any{"method": "item/completed", "params": map[string]any{"completedAtMs": 1, "threadId": threadID, "turnId": turnID, "item": map[string]any{"id": "item-before-auth", "type": "agentMessage", "text": "partial", "phase": "commentary"}}})
				send(map[string]any{"id": "server-auth-1", "method": "account/chatgptAuthTokens/refresh", "params": map[string]any{"reason": "unauthorized"}})
			case "approval-before-attach":
				send(map[string]any{"id": "server-approval-1", "method": "item/commandExecution/requestApproval", "params": fakeCommandApprovalParams(threadID, turnID)})
			case "tool-call-before-attach":
				send(map[string]any{"id": "server-tool-1", "method": protocolv2.MethodItemToolCall, "params": fakeDynamicToolCallParams(threadID, turnID)})
			case "blocking-approval-concurrent":
				if turnCounter == 1 {
					send(map[string]any{"id": "server-approval-1", "method": "item/commandExecution/requestApproval", "params": fakeCommandApprovalParams(threadID, turnID)})
				} else {
					completeTurn(threadID, turnID)
				}
			case "blocking-approval-delayed":
				time.Sleep(25 * time.Millisecond)
				send(map[string]any{"id": "server-approval-1", "method": "item/commandExecution/requestApproval", "params": fakeCommandApprovalParams(threadID, turnID)})
			case "unsupported":
				send(map[string]any{"id": "server-unsupported-1", "method": "unknown/serverRequest", "params": map[string]any{"threadId": threadID, "turnId": turnID, "itemId": "item-unsupported"}})
			case "unsupported-no-turn":
				send(map[string]any{"id": "server-unsupported-1", "method": "unknown/serverRequest", "params": map[string]any{"threadId": threadID, "itemId": "item-unsupported"}})
			case "unknown-notification":
				send(map[string]any{"method": "upstream/newNotification", "params": map[string]any{"threadId": threadID, "turnId": turnID}})
			case "malformed-background-notification":
				send(map[string]any{"method": "skills/changed", "params": map[string]any{"extra": true}})
			case "background-notification":
				send(map[string]any{"method": "skills/changed", "params": map[string]any{}})
				completeTurn(threadID, turnID)
			case "native-notifications":
				send(map[string]any{"method": "item/agentMessage/delta", "params": map[string]any{"threadId": threadID, "turnId": turnID, "itemId": "item-delta", "delta": "native delta"}})
				send(map[string]any{"method": "error", "params": map[string]any{"threadId": threadID, "turnId": turnID, "willRetry": true, "error": map[string]any{"message": "native warning", "codexErrorInfo": "serverOverloaded"}}})
				send(map[string]any{"method": "model/rerouted", "params": map[string]any{"threadId": threadID, "turnId": turnID, "fromModel": "from-model", "toModel": "to-model", "reason": "highRiskCyberActivity"}})
				send(map[string]any{"method": "model/verification", "params": map[string]any{"threadId": threadID, "turnId": turnID, "verifications": []string{string(protocolv2.ModelVerificationTrustedAccessForCyber)}}})
				send(map[string]any{"method": "configWarning", "params": map[string]any{"summary": "native config summary", "details": "native config details", "path": "/tmp/config.toml"}})
				completeTurn(threadID, turnID)
			case "malformed-notification":
				send(map[string]any{"method": "item/agentMessage/delta", "params": map[string]any{"threadId": threadID, "turnId": turnID, "delta": "missing item id"}})
			case "malformed-token-usage-notification":
				send(map[string]any{"method": "thread/tokenUsage/updated", "params": map[string]any{"threadId": threadID, "turnId": turnID, "tokenUsage": map[string]any{"last": fakeTokenUsageBreakdown(3, 1, 2, 1, 5)}}})
			case "malformed-turn-plan-notification":
				send(map[string]any{"method": "turn/plan/updated", "params": map[string]any{"threadId": threadID, "turnId": turnID}})
			case "malformed-hook-started-notification":
				send(map[string]any{"method": "hook/started", "params": map[string]any{"threadId": threadID, "turnId": turnID}})
			case "malformed-hook-completed-notification":
				send(map[string]any{"method": "hook/completed", "params": map[string]any{"run": map[string]any{
					"displayOrder":  1,
					"entries":       []any{},
					"eventName":     "preToolUse",
					"executionMode": "sync",
					"handlerType":   "command",
					"id":            "hook-run-1",
					"scope":         "turn",
					"sourcePath":    "/workspace/.codex/hooks.json",
					"startedAt":     1000,
					"status":        "running",
				}}})
			case "malformed-thread-goal-updated-notification":
				send(map[string]any{"method": "thread/goal/updated", "params": map[string]any{"threadId": threadID}})
			case "malformed-realtime-output-audio-notification":
				send(map[string]any{"method": "thread/realtime/outputAudio/delta", "params": map[string]any{"threadId": threadID}})
			case "malformed-guardian-review-started-notification":
				send(map[string]any{"method": "item/autoApprovalReview/started", "params": map[string]any{
					"review":      map[string]any{"status": "inProgress"},
					"reviewId":    "review-1",
					"startedAtMs": 123,
					"threadId":    threadID,
					"turnId":      turnID,
				}})
			case "malformed-guardian-review-completed-notification":
				send(map[string]any{"method": "item/autoApprovalReview/completed", "params": map[string]any{
					"action":        map[string]any{"command": "ls", "cwd": "/repo", "source": "shell", "type": "command"},
					"completedAtMs": 456,
					"review":        map[string]any{"status": "approved"},
					"reviewId":      "review-1",
					"startedAtMs":   123,
					"threadId":      threadID,
					"turnId":        turnID,
				}})
			case "long-line":
				completeLongTurn(threadID, turnID)
			case "concurrent":
				pendingConcurrent = append(pendingConcurrent, threadID+"|"+turnID)
				if len(pendingConcurrent) == 2 {
					parts2 := strings.Split(pendingConcurrent[1], "|")
					parts1 := strings.Split(pendingConcurrent[0], "|")
					completeTurn(parts2[0], parts2[1])
					completeTurn(parts1[0], parts1[1])
				}
			default:
				completeTurn(threadID, turnID)
			}
		case "turn/interrupt":
			if mode == "turn-interrupt-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{"extra": true}})
				continue
			}
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.TurnInterruptResponse{})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{}})
		case "turn/steer":
			if mode == "turn-steer-malformed-response" {
				send(map[string]any{"id": id, "result": map[string]any{}})
				continue
			}
			if mode == "facade" {
				params, _ := message["params"].(map[string]any)
				turnID, _ := params["expectedTurnId"].(string)
				sendProtocolResult(id, protocolv2.TurnSteerResponse{TurnID: defaultString(turnID, "turn-1")})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"turnId": "turn-1"}})
		case "windowsSandbox/readiness":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.WindowsSandboxReadinessResponse{
					Status: protocolv2.WindowsSandboxReadinessReady,
				})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"status": "ready"}})
		case "windowsSandbox/setupStart":
			if mode == "facade" {
				sendProtocolResult(id, protocolv2.WindowsSandboxSetupStartResponse{Started: true})
				continue
			}
			send(map[string]any{"id": id, "result": map[string]any{"started": false}})
		default:
			send(map[string]any{"id": id, "result": map[string]any{}})
		}
	}
}

func fakeCommandApprovalParams(threadID, turnID string) map[string]any {
	return map[string]any{
		"command":     "echo ok",
		"cwd":         "/tmp",
		"itemId":      "item-approval",
		"reason":      "fake approval",
		"startedAtMs": 123,
		"threadId":    threadID,
		"turnId":      turnID,
	}
}

func sendExactReroute(threadID, turnID, fromModel, toModel string) {
	send(map[string]any{"method": "model/rerouted", "params": map[string]any{
		"threadId":  threadID,
		"turnId":    turnID,
		"fromModel": fromModel,
		"toModel":   toModel,
		"reason":    "highRiskCyberActivity",
	}})
}

func fakeAuthRefreshParams() map[string]any {
	return map[string]any{
		"reason": "unauthorized",
	}
}

func fakeDynamicToolCallParams(threadID, turnID string) map[string]any {
	return map[string]any{
		"arguments": map[string]any{"query": "status"},
		"callId":    "call-tool",
		"threadId":  threadID,
		"tool":      "lookup",
		"turnId":    turnID,
	}
}

func fakeToolUserInputParams(threadID, turnID string) map[string]any {
	return map[string]any{
		"itemId": "item-input",
		"questions": []any{
			map[string]any{
				"header":   "Confirm",
				"id":       "q1",
				"question": "Continue?",
			},
		},
		"threadId": threadID,
		"turnId":   turnID,
	}
}

func fakeMCPElicitationParams(threadID, turnID string) map[string]any {
	return map[string]any{
		"serverName": "mcp-server",
		"threadId":   threadID,
		"turnId":     turnID,
	}
}

func jsonValuePtr(value protocolv2.JSONValue) *protocolv2.JSONValue {
	return &value
}

func sendProtocolResult(id any, result any) {
	raw, err := json.Marshal(result)
	if err != nil {
		panic(err)
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		panic(err)
	}
	send(map[string]any{"id": id, "result": decoded})
}

func facadeThreadStartResponse(threadID string, model string) protocolv2.ThreadStartResponse {
	return protocolv2.ThreadStartResponse{
		ApprovalPolicy:    protocolv2.NewAskForApprovalNever(),
		ApprovalsReviewer: protocolv2.ApprovalsReviewerUser,
		CWD:               "/workspace/facade",
		Model:             defaultString(model, "gpt-facade"),
		ModelProvider:     "openai",
		Sandbox:           protocolv2.NewSandboxPolicyReadOnly(protocolv2.SandboxPolicyReadOnly{}),
		Thread:            facadeThread(threadID, nil),
	}
}

func facadeThreadResumeResponse(threadID string, model string) protocolv2.ThreadResumeResponse {
	return protocolv2.ThreadResumeResponse{
		ApprovalPolicy:    protocolv2.NewAskForApprovalNever(),
		ApprovalsReviewer: protocolv2.ApprovalsReviewerUser,
		CWD:               "/workspace/facade",
		Model:             defaultString(model, "gpt-facade"),
		ModelProvider:     "openai",
		Sandbox:           protocolv2.NewSandboxPolicyReadOnly(protocolv2.SandboxPolicyReadOnly{}),
		Thread:            facadeThread(threadID, nil),
	}
}

func facadeThreadForkResponse(threadID string, parentThreadID string, model string) protocolv2.ThreadForkResponse {
	return protocolv2.ThreadForkResponse{
		ApprovalPolicy:    protocolv2.NewAskForApprovalNever(),
		ApprovalsReviewer: protocolv2.ApprovalsReviewerUser,
		CWD:               "/workspace/facade",
		Model:             defaultString(model, "gpt-facade"),
		ModelProvider:     "openai",
		Sandbox:           protocolv2.NewSandboxPolicyReadOnly(protocolv2.SandboxPolicyReadOnly{}),
		Thread:            facadeThread(threadID, &parentThreadID),
	}
}

func facadeThreadGoal(threadID string, objective string, status protocolv2.ThreadGoalStatus, tokenBudget *protocolv2.Nullable[int64]) protocolv2.ThreadGoal {
	return protocolv2.ThreadGoal{
		CreatedAt:       1000,
		Objective:       objective,
		Status:          status,
		ThreadID:        threadID,
		TimeUsedSeconds: 3,
		TokenBudget:     tokenBudget,
		TokensUsed:      42,
		UpdatedAt:       2000,
	}
}

func facadeThreadTurn(turnID string, items []protocolv2.ThreadItem) protocolv2.Turn {
	view := protocolv2.TurnItemsViewFull
	return protocolv2.Turn{
		CompletedAt: protocolv2.Value(int64(2000)),
		DurationMS:  protocolv2.Value(int64(1000)),
		ID:          turnID,
		Items:       items,
		ItemsView:   &view,
		StartedAt:   protocolv2.Value(int64(1000)),
		Status:      protocolv2.TurnStatusCompleted,
	}
}

func facadeThreadAgentMessageItem(itemID string, text string) protocolv2.ThreadItem {
	return protocolv2.NewThreadItemAgentMessage(protocolv2.ThreadItemAgentMessage{
		ID:   itemID,
		Text: text,
	})
}

func facadeRealtimeAudioChunk() protocolv2.ThreadRealtimeAudioChunk {
	return protocolv2.ThreadRealtimeAudioChunk{
		Data:              "base64",
		ItemID:            protocolv2.Value("item-1"),
		NumChannels:       2,
		SampleRate:        24000,
		SamplesPerChannel: protocolv2.Value(uint32(480)),
	}
}

func facadeConfigReadResponse() protocolv2.ConfigReadResponse {
	return protocolv2.ConfigReadResponse{
		Config: protocolv2.Config{
			Model: protocolv2.Value("gpt-5"),
		},
		Origins: map[string]protocolv2.ConfigLayerMetadata{
			"model": {
				Name: protocolv2.NewConfigLayerSourceUser(protocolv2.ConfigLayerSourceUser{
					File: "/config.toml",
				}),
				Version: "v1",
			},
		},
	}
}

func facadeConfigWriteResponse() protocolv2.ConfigWriteResponse {
	return protocolv2.ConfigWriteResponse{
		FilePath: "/home/user/.codex/config.toml",
		Status:   protocolv2.WriteStatusOk,
		Version:  "v2",
	}
}

func facadeConfigRequirementsReadResponse() protocolv2.ConfigRequirementsReadResponse {
	return protocolv2.ConfigRequirementsReadResponse{
		Requirements: protocolv2.Value(protocolv2.ConfigRequirements{
			FeatureRequirements: protocolv2.Value(map[string]bool{
				"alpha": true,
				"beta":  false,
			}),
			Network: protocolv2.Value(protocolv2.NetworkRequirements{
				Domains: protocolv2.Value(map[string]protocolv2.NetworkDomainPermission{
					"example.com": protocolv2.NetworkDomainPermissionAllow,
				}),
			}),
		}),
	}
}

func facadeExperimentalFeatureListResponse() protocolv2.ExperimentalFeatureListResponse {
	return protocolv2.ExperimentalFeatureListResponse{
		Data: []protocolv2.ExperimentalFeature{{
			Announcement:   protocolv2.Null[string](),
			DefaultEnabled: true,
			Description:    protocolv2.Value("Feature A"),
			DisplayName:    protocolv2.Value("Feature A"),
			Enabled:        false,
			Name:           "feature_a",
			Stage:          protocolv2.ExperimentalFeatureStageBeta,
		}},
		NextCursor: protocolv2.Value("cursor-2"),
	}
}

func facadeCollaborationModeListResponse() protocolv2.CollaborationModeListResponse {
	return protocolv2.CollaborationModeListResponse{
		Data: []protocolv2.CollaborationModeMask{{
			Mode:            protocolv2.Value(protocolv2.ModeKindPlan),
			Model:           protocolv2.Null[string](),
			Name:            "Plan",
			ReasoningEffort: protocolv2.Value(protocolv2.ReasoningEffort("medium")),
		}, {
			Name: "Default",
		}},
	}
}

func facadeExternalAgentConfigDetectResponse() protocolv2.ExternalAgentConfigDetectResponse {
	return protocolv2.ExternalAgentConfigDetectResponse{
		Items: []protocolv2.ExternalAgentConfigMigrationItem{{
			CWD:         protocolv2.Value("/repo"),
			Description: "Import command",
			Details: protocolv2.Value(protocolv2.MigrationDetails{
				Commands: &[]protocolv2.CommandMigration{{
					Name: "build",
				}},
			}),
			ItemType: protocolv2.ExternalAgentConfigMigrationItemTypeCOMMANDS,
		}},
	}
}

func facadeHooksListResponse() protocolv2.HooksListResponse {
	return protocolv2.HooksListResponse{
		Data: []protocolv2.HooksListEntry{{
			CWD: "/repo",
			Errors: []protocolv2.HookErrorInfo{{
				Message: "missing command",
				Path:    "/repo/.codex/hooks.json",
			}},
			Hooks: []protocolv2.HookMetadata{{
				Command:       protocolv2.Null[string](),
				CurrentHash:   "hash-1",
				DisplayOrder:  1,
				Enabled:       true,
				EventName:     protocolv2.HookEventNamePreToolUse,
				HandlerType:   protocolv2.HookHandlerTypeCommand,
				IsManaged:     false,
				Key:           "hook-1",
				Matcher:       protocolv2.Value("shell"),
				PluginID:      protocolv2.Null[string](),
				Source:        protocolv2.HookSourceProject,
				SourcePath:    "/repo/.codex/hooks.json",
				StatusMessage: protocolv2.Value("trusted"),
				TimeoutSec:    10,
				TrustStatus:   protocolv2.HookTrustStatusTrusted,
			}},
			Warnings: []string{"review hook"},
		}},
	}
}

func facadeReviewStartResponse() protocolv2.ReviewStartResponse {
	return protocolv2.ReviewStartResponse{
		ReviewThreadID: "review-thread-1",
		Turn: protocolv2.Turn{
			ID:     "turn-review-1",
			Items:  []protocolv2.ThreadItem{},
			Status: protocolv2.TurnStatusInProgress,
		},
	}
}

func facadeSkillsListResponse() protocolv2.SkillsListResponse {
	return protocolv2.SkillsListResponse{
		Data: []protocolv2.SkillsListEntry{{
			CWD: "/repo",
			Errors: []protocolv2.SkillErrorInfo{{
				Message: "invalid skill",
				Path:    "/repo/.codex/skills/bad/SKILL.md",
			}},
			Skills: []protocolv2.SkillMetadata{{
				Dependencies: protocolv2.Value(protocolv2.SkillDependencies{
					Tools: []protocolv2.SkillToolDependency{{
						Command:     protocolv2.Value("rg"),
						Description: protocolv2.Null[string](),
						Transport:   protocolv2.Null[string](),
						Type:        "command",
						URL:         protocolv2.Null[string](),
						Value:       "rg",
					}},
				}),
				Description: "review docs",
				Enabled:     true,
				Interface: protocolv2.Value(protocolv2.SkillInterface{
					DisplayName:      protocolv2.Value("Review"),
					ShortDescription: protocolv2.Null[string](),
				}),
				Name:             "review",
				Path:             "/repo/.codex/skills/review/SKILL.md",
				Scope:            protocolv2.SkillScopeRepo,
				ShortDescription: protocolv2.Value("Review docs"),
			}},
		}},
	}
}

func facadePluginSummary() protocolv2.PluginSummary {
	availability := protocolv2.PluginAvailabilityAVAILABLE
	keywords := []string{"featured"}
	return protocolv2.PluginSummary{
		AuthPolicy:    protocolv2.PluginAuthPolicyONUSE,
		Availability:  &availability,
		Enabled:       true,
		ID:            "plugin-1",
		InstallPolicy: protocolv2.PluginInstallPolicyAVAILABLE,
		Installed:     true,
		Interface: protocolv2.Value(protocolv2.PluginInterface{
			Capabilities:   []string{"skills"},
			ScreenshotURLs: []string{},
			Screenshots:    []string{},
		}),
		Keywords: &keywords,
		Name:     "plugin-one",
		ShareContext: protocolv2.Value(protocolv2.PluginShareContext{
			RemotePluginID: "remote-1",
			SharePrincipals: protocolv2.Value([]protocolv2.PluginSharePrincipal{{
				Name:          "User One",
				PrincipalID:   "user-1",
				PrincipalType: protocolv2.PluginSharePrincipalTypeUser,
				Role:          protocolv2.PluginSharePrincipalRoleOwner,
			}}),
			ShareURL: protocolv2.Value("https://example.test/share"),
		}),
		Source: protocolv2.NewPluginSourceRemote(),
	}
}

func facadePluginDetail() protocolv2.PluginDetail {
	return protocolv2.PluginDetail{
		AppTemplates: []protocolv2.AppTemplateSummary{},
		Apps: []protocolv2.AppSummary{{
			ID:   "app-1",
			Name: "App One",
		}},
		Hooks: []protocolv2.PluginHookSummary{{
			EventName: protocolv2.HookEventNamePreToolUse,
			Key:       "hook-1",
		}},
		MarketplaceName: "market",
		MarketplacePath: protocolv2.Null[string](),
		MCPServers:      []string{"mcp-1"},
		Skills: []protocolv2.SkillSummary{{
			Description: "review",
			Enabled:     true,
			Interface:   protocolv2.Value(protocolv2.SkillInterface{}),
			Name:        "review",
		}},
		Summary: facadePluginSummary(),
	}
}

func facadeThread(threadID string, forkedFromID *string) protocolv2.Thread {
	forkedFrom := protocolv2.Null[string]()
	if forkedFromID != nil {
		forkedFrom = protocolv2.Value(*forkedFromID)
	}
	return protocolv2.Thread{
		AgentNickname: protocolv2.Null[string](),
		AgentRole:     protocolv2.Null[string](),
		CliVersion:    "0.0.0-test",
		CreatedAt:     1000,
		CWD:           "/workspace/facade",
		Ephemeral:     false,
		ForkedFromID:  forkedFrom,
		GitInfo:       protocolv2.Null[protocolv2.GitInfo](),
		ID:            threadID,
		ModelProvider: "openai",
		Name:          protocolv2.Null[string](),
		Path:          protocolv2.Null[string](),
		Preview:       "facade preview",
		SessionID:     "session-" + threadID,
		Source:        protocolv2.NewSessionSourceAppServer(),
		Status:        protocolv2.NewThreadStatusIdle(),
		ThreadSource:  protocolv2.Value(protocolv2.ThreadSource("user")),
		Turns:         []protocolv2.Turn{},
		UpdatedAt:     2000,
	}
}

func completeTurn(threadID, turnID string) {
	item := map[string]any{"id": "item-" + turnID, "type": "agentMessage", "text": "final-" + turnID, "phase": "final_answer"}
	send(map[string]any{"method": "item/completed", "params": map[string]any{"completedAtMs": 1234, "threadId": threadID, "turnId": turnID, "item": item}})
	send(map[string]any{"method": "thread/tokenUsage/updated", "params": map[string]any{"threadId": threadID, "turnId": turnID, "tokenUsage": map[string]any{"last": fakeTokenUsageBreakdown(3, 1, 2, 1, 5), "total": fakeTokenUsageBreakdown(30, 10, 20, 5, 50)}}})
	send(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": threadID, "turn": map[string]any{"id": turnID, "status": "completed", "items": []map[string]any{item}}}})
}

func fakeTokenUsageBreakdown(inputTokens, cachedInputTokens, outputTokens, reasoningOutputTokens, totalTokens int) map[string]any {
	return map[string]any{
		"cachedInputTokens":     cachedInputTokens,
		"inputTokens":           inputTokens,
		"outputTokens":          outputTokens,
		"reasoningOutputTokens": reasoningOutputTokens,
		"totalTokens":           totalTokens,
	}
}

func completeLongTurn(threadID, turnID string) {
	longText := strings.Repeat("x", 70*1024)
	send(map[string]any{"method": "item/completed", "params": map[string]any{"completedAtMs": 1234, "threadId": threadID, "turnId": turnID, "item": map[string]any{"id": "item-" + turnID, "type": "agentMessage", "text": longText, "phase": "final_answer"}}})
	send(map[string]any{"method": "turn/completed", "params": map[string]any{"threadId": threadID, "turn": map[string]any{"id": turnID, "status": "completed", "items": []map[string]any{}}}})
}

func send(message map[string]any) {
	raw, _ := json.Marshal(message)
	os.Stdout.Write(append(raw, '\n'))
}

func appendRecord(path string, record map[string]any) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	raw, _ := json.Marshal(record)
	file.Write(append(raw, '\n'))
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

func itoa(value int) string {
	return strconv.Itoa(value)
}

type recordingWriteCloser struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *recordingWriteCloser) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *recordingWriteCloser) Close() error {
	return nil
}

func (w *recordingWriteCloser) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]byte(nil), w.buf.Bytes()...)
}

func (w *recordingWriteCloser) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}
