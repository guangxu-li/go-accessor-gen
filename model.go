package main

// StructField represents a field in a struct
type StructField struct {
	Name                 string
	Type                 string
	DeferrencedFieldType string // the type of the field after deferrencing, if type type is a pointer to a primitive
	PrimitivePointer     bool   // true if the field is a pointer to a primitive type
}

// StructInfo holds information about a struct and its fields
type StructInfo struct {
	StructName    string
	Fields        []StructField
	TypeParamsStr string
}

// FileData holds necessary data to generate the target file
type FileData struct {
	PackageName string
	Imports     []string
	Structs     []StructInfo
	Mode        ModeEnum
}
