package protoparser

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
)

// FileDescriptorsFromPaths loads the file descriptors for each .proto file in protoFiles.
// It attempts to infer imports in the .proto files from the file paths in importPaths
func FileDescriptorsFromPaths(importPaths []string, protoFiles []string) ([]*desc.FileDescriptor, error) {
	parser := protoparse.Parser{
		ImportPaths: importPaths,
	}
	descriptors, err := parser.ParseFiles(protoFiles...)
	if err != nil {
		return nil, err
	}
	return descriptors, nil
}
