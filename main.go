package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
)

func main() {
	director := func(r *http.Request) {
		parser := protoparse.Parser{}
		descriptors, err := parser.ParseFiles("hello.proto")
		if err != nil {
			log.Printf("Error parsing proto file: %v", err)
			return
		}

		msg := descriptors[0].FindMessage("hello.Hello")
		if msg == nil {
			log.Printf("Unable to find message: hello.Hello")
			return
		}

		helloMsg := dynamic.NewMessage(msg)
		err = jsonpb.Unmarshal(r.Body, helloMsg)
		if err != nil {
			log.Printf("Unable to unmarshal into json: %v", err)
		}

		reqBytes, err := proto.Marshal(helloMsg)
		if err != nil {
			log.Printf("Unable to marshal message: %v", err)
			return
		}
		buffer := bytes.NewBuffer(reqBytes)
		r.Body = ioutil.NopCloser(buffer)
		r.ContentLength = int64(buffer.Len())
	}
	proxy := &httputil.ReverseProxy{Director: director}
	http.HandleFunc("/", proxy.ServeHTTP)

	log.Fatal(http.ListenAndServe(":7777", nil))
}
