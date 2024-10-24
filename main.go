package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

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
	return processDirectory(options)
}

// processDirectory processes all Go files in a directory, either recursively or non-recursively
func processDirectory(o *Options) error {
	if o.Recursive {
		// Walk the directory recursively
		return filepath.WalkDir(o.Dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if ignoreDirEntry(d) {
				return nil
			}
			if err := processFile(path, o.Mode); err != nil {
				return fmt.Errorf("error processing file %s: %v", path, err)
			}
			return nil
		})
	} else {
		// Read the directory non-recursively
		entries, err := os.ReadDir(o.Dir)
		if err != nil {
			return fmt.Errorf("error reading directory %s: %v", o.Dir, err)
		}

		for _, entry := range entries {
			if ignoreDirEntry(entry) {
				continue
			}
			if err := processFile(filepath.Join(o.Dir, entry.Name()), o.Mode); err != nil {
				return fmt.Errorf("error processing file %s: %v", entry.Name(), err)
			}
		}
		return nil
	}
}

// ignoreDirEntry returns true if the directory entry should be ignored.
func ignoreDirEntry(entry fs.DirEntry) bool {
	return entry.IsDir() ||
		!strings.HasSuffix(entry.Name(), ".go") ||
		strings.HasSuffix(entry.Name(), "_accessor_gen.go") ||
		strings.HasSuffix(entry.Name(), "_test.go")
}

// processFile processes a single Go file, generates getters and setters, and writes them to a new file.
func processFile(filePath string, mode ModeEnum) error {
	// Parse the file
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing file %v: %v", filePath, err)
	}

	// Collect import declarations from the original file
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
				deferrencedFieldType, primitivePointer := PrimitivePointerDeferrence(field)
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
	if err = tmpl.Execute(&output, data); err != nil {
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

func PrimitivePointerDeferrence(field *ast.Field) (string, bool) {
	starExpr, ok := field.Type.(*ast.StarExpr)
	if !ok {
		return "", false
	}
	// Check if the element of the pointer is an identifier (ast.Ident)
	ident, ok := starExpr.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	_, ok = primitiveTypes[ident.Name]
	return ident.Name, ok
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
	default:
		return ""
	}
}
