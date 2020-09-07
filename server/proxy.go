package server

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
)

type Server struct{}

func New() *Server {
	return &Server{}
}

func parseMessageTypes(r *http.Request) (srcMsg, dstMsg string, err error) {
	ctype := r.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(ctype)
	fmt.Println(params)
	if err != nil {
		return "", "", err
	}
	return params["reqmsg"], params["respmsg"], nil
}

func (s *Server) Run() {
	var reqMsg, respMsg string
	director := func(r *http.Request) {
		parser := protoparse.Parser{}
		descriptors, err := parser.ParseFiles("../gen/hello.proto")
		if err != nil {
			log.Printf("Error parsing proto file: %v", err)
			return
		}

		reqMsg, respMsg, err = parseMessageTypes(r)
		if err != nil {
			log.Printf("Error parsing content-type: %v", err)
			return
		}

		msg := descriptors[0].FindMessage(reqMsg)
		if msg == nil {
			log.Printf("Unable to find message: %v", reqMsg)
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

	modifyResp := func(r *http.Response) error {

		parser := protoparse.Parser{}
		descriptors, err := parser.ParseFiles("../gen/hello.proto")
		if err != nil {
			log.Printf("Error parsing proto file: %v", err)
			return err
		}

		msg := descriptors[0].FindMessage(respMsg)
		if msg == nil {
			log.Printf("Unable to find message: %v", respMsg)
			return err
		}

		helloMsg := dynamic.NewMessage(msg)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v", err)
			return err
		}
		err = r.Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
		fmt.Println(string(body))
		err = proto.Unmarshal(body, helloMsg)
		if err != nil {
			log.Printf("Unable to unmarshal into json: %v", err)
			return err
		}

		marshaler := jsonpb.Marshaler{}
		buf := bytes.NewBuffer(nil)
		err = marshaler.Marshal(buf, helloMsg)
		fmt.Println(buf)
		if err != nil {
			log.Printf("Failed to marshal response: %v", err)
			return err
		}
		r.Body = ioutil.NopCloser(buf)
		r.ContentLength = int64(buf.Len())
		r.Header.Set("Content-Length", strconv.Itoa(buf.Len()))
		return nil
	}

	proxy := &httputil.ReverseProxy{Director: director, ModifyResponse: modifyResp}
	http.HandleFunc("/", proxy.ServeHTTP)

	log.Fatal(http.ListenAndServe(":7777", nil))

}
