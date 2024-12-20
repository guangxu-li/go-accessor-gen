package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

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
			if file, ok := astFiles.Load(filename); ok {
				return file.(*ast.File), nil
			}

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

func collectTmplData(node *ast.File, filePath string, mode ModeEnum) *FileData {
	dirPath := filepath.Dir(filePath)
	imports := collectImports(node)

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

			var typeParams []string
			if typeSpec.TypeParams != nil {
				for _, param := range typeSpec.TypeParams.List {
					for _, name := range param.Names {
						typeParams = append(typeParams, name.Name)
					}
				}
			}

			typeParamsStr := ""
			if len(typeParams) > 0 {
				typeParamsStr = "[" + strings.Join(typeParams, ", ") + "]"
			}

			structs = append(structs, StructInfo{
				StructName:    typeSpec.Name.Name,
				Fields:        fields,
				TypeParamsStr: typeParamsStr,
			})
		}
	}

	// If no field found, skip file
	if fieldCnt == 0 {
		return nil
	}

	// Generate the output file content
	return &FileData{
		PackageName: node.Name.Name,
		Imports:     imports,
		Structs:     structs,
		Mode:        mode,
	}
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
