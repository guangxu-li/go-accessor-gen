package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
)

func main() {
	if err := Process(funcOptionsFromFlags()...); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func Process(opts ...FuncOption) error {
	options := FuncOptions(opts).New()
	return process(options)
}

// process processes all Go files in a directory, either recursively or non-recursively
func process(o *Options) error {
	if o.Recursive {
		// Walk the directory recursively
		return filepath.WalkDir(o.Dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				return nil
			}
			return processDir(path, o.Mode)
		})
	} else {
		if err := processDir(o.Dir, o.Mode); err != nil {
			return fmt.Errorf("error processing directory %s: %w", o.Dir, err)
		}
	}
	return nil
}

func processDir(dirPath string, mode ModeEnum) error {
	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("error getting absolute path for %s: %w", dirPath, err)
	}
	resp, err := loadPackages(dirPath)
	if err != nil {
		return fmt.Errorf("error loading packages: %w", err)
	}
	pkgs := resp.packages
	astFiles := resp.astFiles

	for _, pkg := range pkgs {
		for _, filePath := range pkg.GoFiles {
			astFileInterface, ok := astFiles.Load(filePath)
			if !ok {
				return fmt.Errorf("error loading ast file for %s", filePath)
			}
			astFile := astFileInterface.(*ast.File)
			if err := processFile(pkgs, astFile, filePath, mode); err != nil {
				return fmt.Errorf("error processing file %s: %w", filePath, err)
			}
		}
	}

	return nil
}

func processFile(pkgs []*packages.Package, node *ast.File, filePath string, mode ModeEnum) error {
	if ignoreFilePath(filepath.Base(filePath)) {
		return nil
	}

	dirPath := filepath.Dir(filePath)
	imports := collectImports(node)

	// Collect struct information
	var structs []StructInfo
	fieldCnt := 0
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			var fields []StructField
			for _, field := range structType.Fields.List {
				fieldType := exprToString(field.Type)
				deferrencedFieldType := ""
				primitivePointer := isPrimitivePointer(field.Type, dirPath)
				if primitivePointer {
					deferrencedFieldType = fieldType[1:]
				}
				for _, fieldName := range field.Names {
					fieldCnt += 1
					fields = append(fields, StructField{
						Name:                 fieldName.Name,
						Type:                 fieldType,
						DeferrencedFieldType: deferrencedFieldType,
						PrimitivePointer:     primitivePointer,
					})
				}
			}

			structs = append(structs, StructInfo{
				StructName: typeSpec.Name.Name,
				Fields:     fields,
			})
		}
	}

	// If no field found, skip file
	if fieldCnt == 0 {
		return nil
	}

	// Generate the output file content
	data := FileData{
		PackageName: node.Name.Name,
		Imports:     imports,
		Structs:     structs,
		Mode:        mode,
	}
	var output bytes.Buffer
	tmpl := template.Must(
		template.New("accessor").
			Funcs(template.FuncMap{"CapitalizeFirstLetter": CapitalizeFirstLetter}).
			Parse(methodTemplate),
	)
	if err := tmpl.Execute(&output, data); err != nil {
		return fmt.Errorf("error generating output: %v", err)
	}

	// Fix imports and format the code
	formattedSource, err := goImportsAndFormat(output.Bytes(), filePath)
	if err != nil {
		return fmt.Errorf("error formatting file %v: %v", filePath, err)
	}

	// Write the formatted output to a new Go file
	outputFilePath := strings.TrimSuffix(filePath, ".go") + "_accessor_gen.go"
	if strings.HasSuffix(filePath, "_gen.go") {
		outputFilePath = strings.TrimSuffix(filePath, "_gen.go") + "_accessor_gen.go"
	}
	if err := os.WriteFile(outputFilePath, formattedSource, 0o644); err != nil {
		return fmt.Errorf("error writing file %v: %v", outputFilePath, err)
	}

	fmt.Printf("Generated %ss for file: %s\n", mode, filePath)
	return nil
}

// ignoreFilePath returns true if the directory entry should be ignored.
func ignoreFilePath(path string) bool {
	return !strings.HasSuffix(path, ".go") ||
		strings.HasSuffix(path, "_accessor_gen.go") ||
		strings.HasSuffix(path, "_test.go")
}

// isPrimitivePointer checks if a field is a pointer to a primitive type and returns the type name.
func isPrimitivePointer(fieldType ast.Expr, dirPath string) bool {
	starExpr, ok := fieldType.(*ast.StarExpr)
	if !ok {
		return false
	}
	resp, _ := loadPackages(dirPath) // second time call shall read from cache without error
	pkgs := resp.packages

	for _, pkg := range pkgs {
		typ := pkg.TypesInfo.TypeOf(starExpr.X)
		if typ == nil {
			continue
		}

		if _, ok := typ.Underlying().(*types.Basic); ok {
			return true
		}
	}

	return false
}

// loadPackages loads the package with the specific name at the specified directory path with cache.
func loadPackages(dirPath string) (*loadPackagesResponse, error) {
	if result, ok := packageCache[dirPath]; ok {
		return result, nil
	}

	astFiles := &sync.Map{}
	cfg := &packages.Config{
		Mode: packages.LoadAllSyntax,
		Dir:  dirPath,
		ParseFile: func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
			file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
			astFiles.Store(filename, file)
			return file, err
		},
	}
	pkgs, err := packages.Load(cfg)
	if err != nil {
		return nil, fmt.Errorf("error loading package for %s: %w", dirPath, err)
	}
	resp := &loadPackagesResponse{
		packages: pkgs,
		astFiles: astFiles,
	}

	packageCache[dirPath] = resp

	return resp, nil
}

// goImportsAndFormat formats the Go code and fixes imports using the imports.Process function.
func goImportsAndFormat(source []byte, filename string) ([]byte, error) {
	// Use imports.Process from the "golang.org/x/tools/imports" package
	// This will format the code and also fix missing/unused imports
	options := &imports.Options{
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
		FormatOnly: false, // true means only format, false means format and fix imports
	}
	return imports.Process(filename, source, options)
}

// collectImports extracts all import statements from the parsed file.
func collectImports(node *ast.File) (imports []string) {
	for _, imp := range node.Imports {
		str := imp.Path.Value
		if imp.Name != nil {
			str = imp.Name.Name + " " + str // import with alias
		}
		imports = append(imports, str)
	}
	return imports
}

// exprToString converts an expression (field type) to its string representation.
func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	case *ast.IndexExpr:
		return exprToString(t.X) + "[" + exprToString(t.Index) + "]"
	case *ast.IndexListExpr:
		indices := make([]string, len(t.Indices))
		for i, index := range t.Indices {
			indices[i] = exprToString(index)
		}
		return exprToString(t.X) + "[" + strings.Join(indices, ", ") + "]"
	default:
		return ""
	}
}
