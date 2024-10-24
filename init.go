package main

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/tools/go/packages"
)

var cwd string

type loadPackagesResponse struct {
	packages []*packages.Package
	astFiles *sync.Map // key: file path, value: *ast.File
}

var packageCache = make(map[string]*loadPackagesResponse, 10240)

func init() {
	var err error
	if cwd, err = os.Getwd(); err != nil {
		fmt.Printf("Error: could not get current working directory: %v\n", err)
		os.Exit(1)
	}
}
