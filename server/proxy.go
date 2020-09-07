package server

import (
	"bytes"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
)

type Server struct {
	ProtoPath string
	Port      uint16
}

type Config struct {
	ProtoPath string
	Port      uint16
}

func New(cfg Config) *Server {
	return &Server{
		ProtoPath: cfg.ProtoPath,
		Port:      cfg.Port,
	}
}

func parseMessageTypes(r *http.Request) (srcMsg, dstMsg string, err error) {
	ctype := r.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(ctype)
	if err != nil {
		return "", "", err
	}
	return params["reqmsg"], params["respmsg"], nil
}

func fileDescriptorFromProto(file string) (*desc.FileDescriptor, error) {
	parser := protoparse.Parser{}
	descriptors, err := parser.ParseFiles(file)
	if err != nil {
		return nil, err
	}
	return descriptors[0], nil
}

func (s *Server) Run() {
	var reqMsg, respMsg string
	fd, err := fileDescriptorFromProto(s.ProtoPath)
	if err != nil {
		log.Printf("Error parsing protofile: %v", err)
		return
	}
	director := func(r *http.Request) {
		reqMsg, respMsg, err = parseMessageTypes(r)
		if err != nil {
			log.Printf("Error parsing content-type: %v", err)
			return
		}

		msgDescriptor := fd.FindMessage(reqMsg)
		if msgDescriptor == nil {
			log.Printf("Unable to find message: %v", reqMsg)
			return
		}

		msg := dynamic.NewMessage(msgDescriptor)
		err = jsonpb.Unmarshal(r.Body, msg)
		if err != nil {
			log.Printf("Unable to unmarshal into json: %v", err)
			return
		}

		reqBytes, err := proto.Marshal(msg)
		if err != nil {
			log.Printf("Unable to marshal message: %v", err)
			return
		}
		buffer := bytes.NewBuffer(reqBytes)
		r.Body = ioutil.NopCloser(buffer)
		r.ContentLength = int64(buffer.Len())
	}

	modifyResp := func(r *http.Response) error {
		msgDescriptor := fd.FindMessage(respMsg)
		if msgDescriptor == nil {
			log.Printf("Unable to find message: %v", respMsg)
			return err
		}

		msg := dynamic.NewMessage(msgDescriptor)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read response body: %v", err)
			return err
		}
		err = r.Body.Close()
		if err != nil {
			log.Printf("Error closing body: %v", err)
		}
		err = proto.Unmarshal(body, msg)
		if err != nil {
			log.Printf("Unable to unmarshal into json: %v", err)
			return err
		}

		marshaler := jsonpb.Marshaler{}
		buf := bytes.NewBuffer(nil)
		err = marshaler.Marshal(buf, msg)
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
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(s.Port)), nil))
}
