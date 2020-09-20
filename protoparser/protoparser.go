package protoparser

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
)

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
