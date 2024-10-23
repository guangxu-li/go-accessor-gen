package main

// StructField represents a field in a struct
type StructField struct {
	Name string
	Type string
}

// StructInfo holds information about a struct and its fields
type StructInfo struct {
	StructName string
	Fields     []StructField
}

// FileData holds necessary data to generate the target file
type FileData struct {
	PackageName string
	Imports     []string
	Structs     []StructInfo
	Mode        ModeEnum
}
