package main

import (
	"fmt"
	"os"
)

var cwd string

var primitiveTypes = map[string]struct{}{
	"int":        {},
	"int8":       {},
	"int16":      {},
	"int32":      {},
	"int64":      {},
	"uint":       {},
	"uint8":      {},
	"uint16":     {},
	"uint32":     {},
	"uint64":     {},
	"uintptr":    {},
	"float32":    {},
	"float64":    {},
	"complex64":  {},
	"complex128": {},
	"bool":       {},
	"string":     {},
	"byte":       {},
	"rune":       {},
}

func init() {
	var err error
	if cwd, err = os.Getwd(); err != nil {
		fmt.Printf("Error: could not get current working directory: %v\n", err)
		os.Exit(1)
	}
}
