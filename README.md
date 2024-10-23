# Accessor Generator

This tool generates **getters** and **setters** for struct fields in Go. You can specify whether to generate:

- **Only getters**
- **Only setters**
- **Both getters and setters** (referred to as "accessors").

## Features

- Generate **getters** and/or **setters** for all structs in your Go files.
- Optionally process directories **recursively**.
- Control whether to generate **only getters**, **only setters**, or **both** via the `--mode` flag.

## Requirements

- **Go 1.22+** is required to use this tool.

## Installation

To install the tool globally using `go install`, simply run:

```bash
go install github.com/guangxu-li/go-generator-gen@latest
```

This will install the `accessor-gen` binary to your `$GOPATH/bin` or `$HOME/go/bin`, which is typically included in your system's `PATH`. After installation, you can run the tool directly from the command line.

For example:

```bash
accessor-gen --dir <directory> --mode <mode> [--recursive]
```

Alternatively, if you prefer cloning the repository:

```bash
git clone https://github.com/guangxu-li/go-generator-gen.git
cd accessor-generator
go mod tidy
go build -o accessor-gen
```

## Usage

You can run the generator directly from the command line by specifying a directory. The tool scans the specified directory for Go files and generates the corresponding accessors.

```bash
accessor-gen --dir <directory> --mode <mode> [--recursive]
```

### Flags

- `--dir`: The directory to process. If not provided, the current directory is used.
- `--mode`: Specifies what to generate. It can be one of:
    - `getter`: Only generate getters.
    - `setter`: Only generate setters.
    - `accessor`: Generate both getters and setters (default).
- `--recursive`: If provided, the tool will process directories recursively.

### Examples

#### Example 1: Generate both getters and setters (default mode)

```bash
accessor-gen --dir ./models
```

This will process all Go files in the `./models` directory and generate both getters and setters for all structs.

#### Example 2: Generate only getters

```bash
accessor-gen --dir ./models --mode getter
```

This will generate only getters for all structs in the `./models` directory.

#### Example 3: Generate only setters recursively

```bash
accessor-gen --dir ./models --mode setter --recursive
```

This will generate only setters for all structs in the `./models` directory and all of its subdirectories.

## Output

The generated getters and/or setters are written to new `.go` files with the suffix `_accessor_gen.go`. These files are placed in the same directory as the original Go files but are excluded from being overwritten during subsequent runs.

For example, if you have a file called `model.go` in the `./models` directory, the generated accessors will be written to `model_accessor_gen.go`.

## Contributing

If you'd like to contribute or report issues, please submit a pull request or open an issue on the repository.

## License

This project is licensed under the MIT License. See the `LICENSE` file for more details.
