package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

//          ┌─────────────────────────────────────────────────────────┐
//          │                          Flag                           │
//          └─────────────────────────────────────────────────────────┘

// Function to display usage information
func printUsage() {
	fmt.Printf("Usage: %s [options]\n", os.Args[0])
	fmt.Println("Options:")
	fmt.Println("  --dir        Directory to process (default is current working directory)")
	fmt.Println("  --mode       Mode to generate: 'getter', 'setter', or 'accessor' (default: accessor)")
	fmt.Println("  --recursive  Recursively process directories (default: false)")
	fmt.Println("  --version    Show version information")
	fmt.Println("  --help       Show this help message")
}

func funcOptionsFromFlags() FuncOptions {
	dirFlag := flag.String("dir", cwd, "directory to process")
	modeFlag := flag.String("mode", ModeAccessor.String(), "getter, setter, accessor")
	recursiveFlag := flag.Bool("recursive", false, "process directory recursively")
	versionFlag := flag.Bool("version", false, "display version information")

	flag.Usage = printUsage
	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	return FuncOptions{
		Dir(*dirFlag),
		Mode(*modeFlag),
		Recursive(*recursiveFlag),
	}
}

//          ┌─────────────────────────────────────────────────────────┐
//          │                       Func Option                       │
//          └─────────────────────────────────────────────────────────┘

// Definition
// ────────────────────────────────────────────────────────────────────────────────

type ModeEnum string

func (m ModeEnum) String() string {
	return string(m)
}

const (
	ModeUnknown  ModeEnum = ""
	ModeGetter   ModeEnum = "getter"
	ModeSetter   ModeEnum = "setter"
	ModeAccessor ModeEnum = "accessor"
)

type Options struct {
	Dir       string   // Default is the current working directory.
	Mode      ModeEnum // Default is accessor.
	Recursive bool     // Default is false.
}

type FuncOptions []FuncOption

func (fs FuncOptions) New() *Options {
	options := &Options{
		Dir:       cwd,
		Mode:      ModeAccessor,
		Recursive: false,
	}
	for _, f := range fs {
		f(options)
	}

	return options
}

type FuncOption func(o *Options)

// Options
// ────────────────────────────────────────────────────────────────────────────────

// Dir sets the directory option. Default is the current working directory.
func Dir(dir string) FuncOption {
	return func(o *Options) {
		o.Dir = filepath.Clean(dir)
	}
}

// Mode sets the mode option. Default is accessor.
func Mode[T ~string](m T) FuncOption {
	return func(o *Options) {
		o.Mode = ModeEnum(m)
	}
}

// Recursive sets the recursive option. Default is false.
func Recursive(recursive bool) FuncOption {
	return func(o *Options) {
		o.Recursive = recursive
	}
}

// EnableGetters enables generation of getters without overriding the previous mode.
func EnableGetters() FuncOption {
	return func(o *Options) {
		switch o.Mode {
		case ModeSetter:
			o.Mode = ModeAccessor
		default:
			o.Mode = ModeGetter
		}
	}
}

// EnableSetters enables generation of setters without overriding the previous mode.
func EnableSetters() FuncOption {
	return func(o *Options) {
		switch o.Mode {
		case ModeGetter:
			o.Mode = ModeAccessor
		default:
			o.Mode = ModeSetter
		}
	}
}

// DisableGetters disables generation of getters without overriding the previous mode.
func DisableGetters() FuncOption {
	return func(o *Options) {
		switch o.Mode {
		case ModeAccessor:
			o.Mode = ModeSetter
		default:
			o.Mode = ModeUnknown
		}
	}
}

// DisableSetters disables generation of setters without overriding the previous mode.
func DisableSetters() FuncOption {
	return func(o *Options) {
		switch o.Mode {
		case ModeAccessor:
			o.Mode = ModeGetter
		default:
			o.Mode = ModeUnknown
		}
	}
}
