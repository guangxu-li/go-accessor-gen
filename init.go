package main

import (
	"fmt"
	"os"
)

var cwd string

func init() {
	var err error
	if cwd, err = os.Getwd(); err != nil {
		fmt.Printf("Error: could not get current working directory: %v\n", err)
		os.Exit(1)
	}
}
