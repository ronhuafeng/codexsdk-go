package protocolv2

import "testing"

func TestLookupMethod(t *testing.T) {
	info, ok := LookupMethod(MethodTurnStart)
	if !ok {
		t.Fatal("LookupMethod did not find turn/start")
	}
	if info.Method != MethodTurnStart {
		t.Fatalf("Method = %q, want %q", info.Method, MethodTurnStart)
	}
	if info.Direction != MethodDirectionClientToServer {
		t.Fatalf("Direction = %q, want %q", info.Direction, MethodDirectionClientToServer)
	}
	if info.Kind != MethodKindRequest {
		t.Fatalf("Kind = %q, want %q", info.Kind, MethodKindRequest)
	}
	if info.ResponseSchema != "v2/TurnStartResponse.json" || info.ResponseSchemaStatus != ResponseSchemaStatusDeclared {
		t.Fatalf("unexpected response schema facts: %#v", info)
	}
	if info.Stability != MethodStabilityStable {
		t.Fatalf("Stability = %q, want %q", info.Stability, MethodStabilityStable)
	}

	experimental, ok := LookupMethod(MethodThreadRealtimeStart)
	if !ok {
		t.Fatal("LookupMethod did not find thread/realtime/start")
	}
	if experimental.Stability != MethodStabilityExperimental {
		t.Fatalf("thread/realtime/start stability = %q, want %q", experimental.Stability, MethodStabilityExperimental)
	}
}

func TestLookupMethodUnknown(t *testing.T) {
	if _, ok := LookupMethod("not/a/method"); ok {
		t.Fatal("LookupMethod accepted unknown method")
	}
}

func TestAllMethodsSortedAndComplete(t *testing.T) {
	methods := AllMethods()
	if len(methods) != 201 {
		t.Fatalf("AllMethods length = %d, want 201", len(methods))
	}
	for i := 1; i < len(methods); i++ {
		if methods[i-1].Method >= methods[i].Method {
			t.Fatalf("AllMethods is not strictly sorted at %d: %q >= %q", i, methods[i-1].Method, methods[i].Method)
		}
	}
}
