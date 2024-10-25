package main

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"

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
		return processRecursively(o.Dir, o.Mode)
	}
	return processNonRecursively(o.Dir, o.Mode)
}

func processRecursively(dir string, mode ModeEnum) error {
	if err := processNonRecursively(dir, mode); err != nil {
		return err
	}

	entrys, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error reading directory %s: %w", dir, err)
	}

	for _, e := range entrys {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if err := processRecursively(path, mode); err != nil {
			return err
		}
	}
	return nil
}

func processNonRecursively(dir string, mode ModeEnum) error {
	dirPath, err := filepath.Abs(dir)
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
			if err := processFile(astFile, filePath, mode); err != nil {
				return fmt.Errorf("error processing file %s: %w", filePath, err)
			}
		}
	}

	return nil
}

func processFile(node *ast.File, filePath string, mode ModeEnum) error {
	if ignoreFile(filepath.Base(filePath)) {
		return nil
	}

	data := collectTmplData(node, filePath, mode)
	if data == nil {
		return nil
	}

	bytes, err := executeTmpl(data, filePath)
	if err != nil {
		return fmt.Errorf("error generating tmpl for %s: %w", filePath, err)
	}

	if bytes, err = goImportsAndFormat(bytes, filePath); err != nil {
		return fmt.Errorf("error formatting file %v: %v", filePath, err)
	}

	if err := writeToFile(bytes, filePath); err != nil {
		return fmt.Errorf("error writing file %v: %v", filePath, err)
	}

	fmt.Printf("Generated %ss for file: %s\n", mode, filePath)
	return nil
}

func writeToFile(bytes []byte, filePath string) error {
	outputFilePath := strings.TrimSuffix(filePath, ".go") + "_accessor_gen.go"
	if strings.HasSuffix(filePath, "_gen.go") {
		outputFilePath = strings.TrimSuffix(filePath, "_gen.go") + "_accessor_gen.go"
	}
	if err := os.WriteFile(outputFilePath, bytes, 0o644); err != nil {
		return fmt.Errorf("error writing file %v: %v", outputFilePath, err)
	}
	return nil
}

// ignoreFile returns true if the directory entry should be ignored.
func ignoreFile(path string) bool {
	return !strings.HasSuffix(path, ".go") ||
		strings.HasSuffix(path, "_accessor_gen.go") ||
		strings.HasSuffix(path, "_test.go")
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
