package codexsdk

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestHandwrittenPublicAPI(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	loader := &sdkSourceImporter{root: root, fset: token.NewFileSet(), cache: map[string]*types.Package{}}
	pkg, err := loader.Import("github.com/ronhuafeng/codexsdk-go/codexsdk")
	if err != nil {
		t.Fatal(err)
	}
	declarations := handwrittenDeclarations(loader.fset, pkg)
	sort.Strings(declarations)
	actual := strings.Join(declarations, "\n") + "\n"
	path := filepath.Join(root, "testdata", "handwritten-api.txt")
	if os.Getenv("UPDATE_HANDWRITTEN_API") == "1" {
		if err := os.WriteFile(path, []byte(actual), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if actual != string(want) {
		t.Fatalf("handwritten API changed; update the normative plan and review the canonical allowlist:\n%s", actual)
	}
}

type sdkSourceImporter struct {
	root     string
	fset     *token.FileSet
	cache    map[string]*types.Package
	compiled types.Importer
}

func (i *sdkSourceImporter) Import(path string) (*types.Package, error) {
	if pkg := i.cache[path]; pkg != nil {
		return pkg, nil
	}
	const module = "github.com/ronhuafeng/codexsdk-go/codexsdk"
	if path != module && !strings.HasPrefix(path, module+"/") {
		if i.compiled == nil {
			i.compiled = importer.ForCompiler(i.fset, "gc", i.openExport)
		}
		return i.compiled.Import(path)
	}
	dir := i.root
	if path != module {
		dir = filepath.Join(i.root, strings.TrimPrefix(path, module+"/"))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []*ast.File
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(i.fset, filepath.Join(dir, entry.Name()), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	config := types.Config{Importer: i}
	pkg, err := config.Check(path, i.fset, files, nil)
	if err != nil {
		return nil, err
	}
	i.cache[path] = pkg
	return pkg, nil
}

func (i *sdkSourceImporter) openExport(path string) (io.ReadCloser, error) {
	command := exec.Command("go", "list", "-export", "-json", path)
	command.Env = append(os.Environ(), "GOWORK=off")
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("go list %s: %w", path, err)
	}
	var listed struct{ Export string }
	if err := json.Unmarshal(output, &listed); err != nil {
		return nil, err
	}
	return os.Open(listed.Export)
}

func handwrittenDeclarations(fset *token.FileSet, pkg *types.Package) []string {
	qualifier := func(other *types.Package) string { return other.Path() }
	var declarations []string
	for _, name := range pkg.Scope().Names() {
		object := pkg.Scope().Lookup(name)
		if !object.Exported() || generatedPosition(fset, object.Pos()) {
			continue
		}
		declarations = append(declarations, publicObjectString(object, qualifier))
		typeName, ok := object.(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := typeName.Type().(*types.Named)
		if !ok {
			continue
		}
		methods := types.NewMethodSet(types.NewPointer(named))
		for index := 0; index < methods.Len(); index++ {
			method := methods.At(index).Obj()
			if method.Exported() && !generatedPosition(fset, method.Pos()) {
				declarations = append(declarations, fmt.Sprintf("method %s.%s.%s%s", pkg.Path(), named.Obj().Name(), method.Name(), types.TypeString(method.Type(), qualifier)))
			}
		}
	}
	return declarations
}

func publicObjectString(object types.Object, qualifier types.Qualifier) string {
	typeName, ok := object.(*types.TypeName)
	if !ok {
		return types.ObjectString(object, qualifier)
	}
	named, ok := typeName.Type().(*types.Named)
	if !ok {
		return types.ObjectString(object, qualifier)
	}
	structure, ok := named.Underlying().(*types.Struct)
	if !ok {
		return types.ObjectString(object, qualifier)
	}
	for index := 0; index < structure.NumFields(); index++ {
		if structure.Field(index).Exported() {
			return types.ObjectString(object, qualifier)
		}
	}
	return fmt.Sprintf("type %s.%s struct{ /* unexported fields */ }", object.Pkg().Path(), object.Name())
}

func TestPublicObjectStringMasksOnlyStructsWithoutExportedFields(t *testing.T) {
	pkg := types.NewPackage("example.com/inventory", "inventory")
	hidden := types.NewField(token.NoPos, pkg, "hidden", types.Typ[types.String], false)
	exported := types.NewField(token.NoPos, pkg, "Visible", types.Typ[types.String], false)

	opaqueName := types.NewTypeName(token.NoPos, pkg, "Opaque", nil)
	types.NewNamed(opaqueName, types.NewStruct([]*types.Var{hidden}, nil), nil)
	if got := publicObjectString(opaqueName, nil); !strings.Contains(got, "unexported fields") {
		t.Fatalf("opaque struct inventory = %q, want masked private layout", got)
	}

	publicName := types.NewTypeName(token.NoPos, pkg, "Public", nil)
	types.NewNamed(publicName, types.NewStruct([]*types.Var{hidden, exported}, nil), nil)
	got := publicObjectString(publicName, nil)
	if !strings.Contains(got, "Visible string") || strings.Contains(got, "unexported fields") {
		t.Fatalf("public struct inventory = %q, want exported field retained", got)
	}
}

func generatedPosition(fset *token.FileSet, position token.Pos) bool {
	filename := filepath.ToSlash(fset.Position(position).Filename)
	return strings.HasSuffix(filename, ".gen.go") || strings.Contains(filename, "/protocolv2/")
}
