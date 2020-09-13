package protoparser

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
)

func FileDescriptorFromProto(file string) (*desc.FileDescriptor, error) {
	parser := protoparse.Parser{}
	descriptors, err := parser.ParseFiles(file)
	if err != nil {
		return nil, err
	}
	return descriptors[0], nil
}
