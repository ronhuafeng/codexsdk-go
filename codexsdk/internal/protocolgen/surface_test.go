package protocolgen

import (
	"strings"
	"testing"
)

func TestClassifyExportedSurfaceDerivesMixedTypeFromMembers(t *testing.T) {
	stable := []byte(`package protocolv2
type Event struct { ID string }
type State string
const StateReady State = "ready"
func (Event) MarshalJSON() ([]byte, error) { return nil, nil }
`)
	complete := []byte(`package protocolv2
type Event struct { ID string; Preview string }
type State string
const (
	StateReady State = "ready"
	StatePreview State = "preview"
)
func (Event) MarshalJSON() ([]byte, error) { return nil, nil }
`)

	got, err := ClassifyExportedSurface(stable, complete)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Stability{
		"Event":             StabilityMixed,
		"Event.ID":          StabilityStable,
		"Event.Preview":     StabilityExperimental,
		"Event.MarshalJSON": StabilityStable,
		"State":             StabilityMixed,
		"StateReady":        StabilityStable,
		"StatePreview":      StabilityExperimental,
	}
	if len(got) != len(want) {
		t.Fatalf("surface length = %d, want %d: %#v", len(got), len(want), got)
	}
	for _, entry := range got {
		if want[entry.Name] != entry.Stability {
			t.Errorf("%s %s stability = %q, want %q", entry.Kind, entry.Name, entry.Stability, want[entry.Name])
		}
	}
	for _, entry := range got {
		if entry.Name == "StatePreview" && entry.Owner != "State" {
			t.Errorf("StatePreview owner = %q, want State", entry.Owner)
		}
	}
}

func TestClassifyExportedPackageParsesGeneratedFilesSeparately(t *testing.T) {
	stable := [][]byte{
		[]byte("package protocolv2\ntype Event struct{}\n"),
		[]byte("package protocolv2\nconst Stable = 1\n"),
	}
	complete := append(append([][]byte{}, stable...), []byte("package protocolv2\nconst Preview = 2\n"))

	got, err := ClassifyExportedPackage(stable, complete)
	if err != nil {
		t.Fatal(err)
	}
	classified := map[string]Stability{}
	for _, entry := range got {
		classified[entry.Name] = entry.Stability
	}
	if classified["Stable"] != StabilityStable || classified["Preview"] != StabilityExperimental {
		t.Fatalf("classifications = %#v", classified)
	}
}

func TestClassifyExportedSurfaceClassifiesUnionVariantValues(t *testing.T) {
	stable := []byte(`package protocolv2
type ItemKind string
const ItemKindMessage ItemKind = "message"
`)
	complete := []byte(`package protocolv2
type ItemKind string
const (
	ItemKindMessage ItemKind = "message"
	ItemKindPreview ItemKind = "preview"
)
`)

	got, err := ClassifyExportedSurface(stable, complete)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]SurfaceEntry{}
	for _, entry := range got {
		byName[entry.Name] = entry
	}
	if byName["ItemKind"].Stability != StabilityMixed {
		t.Fatalf("ItemKind = %#v, want mixed owner", byName["ItemKind"])
	}
	preview := byName["ItemKindPreview"]
	if preview.Kind != SurfaceValue || preview.Owner != "ItemKind" || preview.Stability != StabilityExperimental {
		t.Fatalf("ItemKindPreview = %#v", preview)
	}
}

func TestVerifyExportedSurfaceRejectsUnclassifiedExport(t *testing.T) {
	source := []byte("package protocolv2\ntype Event struct { ID string }\n")
	err := VerifyExportedSurface(source, []SurfaceEntry{{Kind: SurfaceType, Name: "Event", Stability: StabilityStable}})
	if err == nil || !strings.Contains(err.Error(), `field "Event.ID" is unclassified`) {
		t.Fatalf("error = %v, want unclassified field", err)
	}
}

func TestValidateSurfaceRejectsMixedMember(t *testing.T) {
	err := ValidateSurface([]SurfaceEntry{{Kind: SurfaceField, Name: "Event.ID", Stability: StabilityMixed}})
	if err == nil || !strings.Contains(err.Error(), "cannot be mixed") {
		t.Fatalf("error = %v, want mixed member rejection", err)
	}
}
