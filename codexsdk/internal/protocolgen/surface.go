package protocolgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
)

type Stability string

const (
	StabilityStable       Stability = "stable"
	StabilityExperimental Stability = "experimental"
	StabilityMixed        Stability = "mixed"
)

type SurfaceKind string

const (
	SurfaceConst     SurfaceKind = "const"
	SurfaceField     SurfaceKind = "field"
	SurfaceFunc      SurfaceKind = "func"
	SurfaceInterface SurfaceKind = "interface_method"
	SurfaceMethod    SurfaceKind = "method"
	SurfaceType      SurfaceKind = "type"
	SurfaceValue     SurfaceKind = "value"
	SurfaceVar       SurfaceKind = "var"
)

// ClassifyExportedSurface compares generated package sources. Stable source is
// the package generated without experimental schema visibility; complete source
// is generated with it. Every exported complete identity receives one entry.
func ClassifyExportedSurface(stableSource, completeSource []byte) ([]SurfaceEntry, error) {
	return ClassifyExportedPackage([][]byte{stableSource}, [][]byte{completeSource})
}

func ClassifyExportedPackage(stableSources, completeSources [][]byte) ([]SurfaceEntry, error) {
	stable, err := exportedPackageSurface(stableSources)
	if err != nil {
		return nil, fmt.Errorf("parse stable generated source: %w", err)
	}
	complete, err := exportedPackageSurface(completeSources)
	if err != nil {
		return nil, fmt.Errorf("parse complete generated source: %w", err)
	}

	entries := make([]SurfaceEntry, 0, len(complete))
	for key, item := range complete {
		stability := StabilityExperimental
		if stableItem, ok := stable[key]; ok && stableItem.Kind == item.Kind {
			stability = StabilityStable
		}
		entries = append(entries, SurfaceEntry{Name: item.Name, Kind: item.Kind, Owner: item.Owner, Stability: stability})
	}
	markMixedOwners(entries)
	sortSurface(entries)
	if err := ValidateSurface(entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func ValidateSurface(entries []SurfaceEntry) error {
	if len(entries) == 0 {
		return fmt.Errorf("surface has no exported identities")
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		key := string(entry.Kind) + "\x00" + entry.Name
		if entry.Name == "" {
			return fmt.Errorf("surface entry has no name")
		}
		switch entry.Kind {
		case SurfaceConst, SurfaceField, SurfaceFunc, SurfaceInterface, SurfaceMethod, SurfaceType, SurfaceValue, SurfaceVar:
		default:
			return fmt.Errorf("surface entry %q has unsupported kind %q", entry.Name, entry.Kind)
		}
		switch entry.Stability {
		case StabilityStable, StabilityExperimental:
		case StabilityMixed:
			if entry.Kind != SurfaceType {
				return fmt.Errorf("non-type surface entry %q cannot be mixed", entry.Name)
			}
		default:
			return fmt.Errorf("surface entry %q is unclassified", entry.Name)
		}
		if seen[key] {
			return fmt.Errorf("surface identity %s %q appears more than once", entry.Kind, entry.Name)
		}
		seen[key] = true
	}
	return nil
}

func VerifyExportedSurface(completeSource []byte, entries []SurfaceEntry) error {
	exported, err := exportedPackageSurface([][]byte{completeSource})
	if err != nil {
		return err
	}
	classified := map[string]bool{}
	for _, entry := range entries {
		classified[string(entry.Kind)+"\x00"+entry.Name] = true
	}
	for key, item := range exported {
		if !classified[key] {
			return fmt.Errorf("exported generated %s %q is unclassified", item.Kind, item.Name)
		}
	}
	for key := range classified {
		if _, ok := exported[key]; !ok {
			return fmt.Errorf("classified generated identity %q is not exported", key)
		}
	}
	return nil
}

func markMixedOwners(entries []SurfaceEntry) {
	owners := map[string]map[Stability]bool{}
	for _, entry := range entries {
		owner := memberOwner(entry)
		if owner == "" {
			continue
		}
		if owners[owner] == nil {
			owners[owner] = map[Stability]bool{}
		}
		owners[owner][entry.Stability] = true
	}
	for i := range entries {
		if entries[i].Kind == SurfaceType && owners[entries[i].Name][StabilityStable] && owners[entries[i].Name][StabilityExperimental] {
			entries[i].Stability = StabilityMixed
		}
	}
}

func memberOwner(entry SurfaceEntry) string {
	if entry.Owner != "" {
		return entry.Owner
	}
	if entry.Kind != SurfaceField && entry.Kind != SurfaceInterface && entry.Kind != SurfaceMethod {
		return ""
	}
	for i := 0; i < len(entry.Name); i++ {
		if entry.Name[i] == '.' {
			return entry.Name[:i]
		}
	}
	return ""
}

type exportedIdentity struct {
	Kind  SurfaceKind
	Name  string
	Owner string
}

func exportedPackageSurface(sources [][]byte) (map[string]exportedIdentity, error) {
	result := map[string]exportedIdentity{}
	for index, source := range sources {
		items, err := exportedFileSurface(source, fmt.Sprintf("generated_%d.go", index))
		if err != nil {
			return nil, err
		}
		for key, item := range items {
			if previous, ok := result[key]; ok && previous != item {
				return nil, fmt.Errorf("exported identity %q has conflicting declarations", item.Name)
			}
			result[key] = item
		}
	}
	return result, nil
}

func exportedFileSurface(source []byte, filename string) (map[string]exportedIdentity, error) {
	file, err := parser.ParseFile(token.NewFileSet(), filename, source, 0)
	if err != nil {
		return nil, err
	}
	result := map[string]exportedIdentity{}
	add := func(kind SurfaceKind, name, owner string) {
		item := exportedIdentity{Kind: kind, Name: name, Owner: owner}
		result[string(kind)+"\x00"+name] = item
	}
	for _, decl := range file.Decls {
		switch node := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				switch item := spec.(type) {
				case *ast.TypeSpec:
					if !item.Name.IsExported() {
						continue
					}
					add(SurfaceType, item.Name.Name, "")
					collectTypeMembers(add, item.Name.Name, item.Type)
				case *ast.ValueSpec:
					kind := SurfaceVar
					if node.Tok == token.CONST {
						kind = SurfaceConst
					}
					owner := expressionName(item.Type)
					if node.Tok == token.CONST && owner != "" {
						kind = SurfaceValue
					}
					for _, name := range item.Names {
						if name.IsExported() {
							add(kind, name.Name, owner)
						}
					}
				}
			}
		case *ast.FuncDecl:
			if !node.Name.IsExported() {
				continue
			}
			if node.Recv == nil {
				add(SurfaceFunc, node.Name.Name, "")
				continue
			}
			if owner := receiverName(node.Recv.List[0].Type); owner != "" && ast.IsExported(owner) {
				add(SurfaceMethod, owner+"."+node.Name.Name, owner)
			}
		}
	}
	return result, nil
}

func collectTypeMembers(add func(SurfaceKind, string, string), owner string, expression ast.Expr) {
	switch node := expression.(type) {
	case *ast.StructType:
		for _, field := range node.Fields.List {
			for _, name := range field.Names {
				if name.IsExported() {
					add(SurfaceField, owner+"."+name.Name, owner)
				}
			}
		}
	case *ast.InterfaceType:
		for _, field := range node.Methods.List {
			for _, name := range field.Names {
				if name.IsExported() {
					add(SurfaceInterface, owner+"."+name.Name, owner)
				}
			}
		}
	}
}

func expressionName(expression ast.Expr) string {
	if expression == nil {
		return ""
	}
	return receiverName(expression)
}

func receiverName(expression ast.Expr) string {
	switch node := expression.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.StarExpr:
		return receiverName(node.X)
	default:
		return ""
	}
}

func sortSurface(entries []SurfaceEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Kind < entries[j].Kind
	})
}
