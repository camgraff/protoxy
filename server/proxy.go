package server

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/dynamic"
)

type Server struct {
	Proxy     *httputil.ReverseProxy
	Port      uint16
	ProtoPath string
}

type Config struct {
	ProtoPath string
	Port      uint16
}

//TODO: Global variables are no bueno
var reqMsg, respMsg, qs string
var fd *desc.FileDescriptor

func New(cfg Config) *Server {
	// Generate file descriptors from proto files
	//TODO: this should be done before the server starts in the cmd package
	var err error
	fd, err = fileDescriptorFromProto(cfg.ProtoPath)
	if err != nil {
		log.Printf("Error parsing protofile: %v", err)
		return nil
	}

	modifyResp := func(r *http.Response) error {
		msgDescriptor := fd.FindMessage(respMsg)
		if msgDescriptor == nil {
			return fmt.Errorf("Unable to find message: %v", respMsg)
		}

		msg := dynamic.NewMessage(msgDescriptor)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("Failed to read response body: %v", err)
		}
		err = r.Body.Close()
		if err != nil {
			return fmt.Errorf("Error closing body: %v", err)
		}
		err = proto.Unmarshal(body, msg)
		if err != nil {
			return fmt.Errorf("Unable to unmarshal into json: %v", err)
		}

		marshaler := jsonpb.Marshaler{}
		buf := bytes.NewBuffer(nil)
		err = marshaler.Marshal(buf, msg)
		if err != nil {
			return fmt.Errorf("Failed to marshal response: %v", err)
		}
		r.Body = ioutil.NopCloser(buf)
		r.ContentLength = int64(buf.Len())
		r.Header.Set("Content-Length", strconv.Itoa(buf.Len()))
		return nil
	}

	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	proxy := &httputil.ReverseProxy{
		Director:       func(*http.Request) {},
		ModifyResponse: modifyResp,
		ErrorHandler:   errorHandler,
	}

	return &Server{
		Proxy:     proxy,
		Port:      cfg.Port,
		ProtoPath: cfg.ProtoPath,
	}
}

func parseMessageTypes(r *http.Request) (srcMsg, dstMsg, qs string, err error) {
	ctype := r.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(ctype)
	if err != nil {
		return "", "", "", err
	}
	return params["reqmsg"], params["respmsg"], params["qs"], nil
}

func fileDescriptorFromProto(file string) (*desc.FileDescriptor, error) {
	parser := protoparse.Parser{}
	descriptors, err := parser.ParseFiles(file)
	if err != nil {
		return nil, err
	}
	return descriptors[0], nil
}

func (s *Server) proxyRequest(w http.ResponseWriter, r *http.Request) {
	var err error
	reqMsg, respMsg, qs, err = parseMessageTypes(r)
	if err != nil {
		errMsg := fmt.Sprintf("Error parsing content-type: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg))
		return
	}

	msgDescriptor := fd.FindMessage(reqMsg)
	if msgDescriptor == nil {
		errMsg := fmt.Sprintf("Unable to find message: %v", reqMsg)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg))
		return
	}

	msg := dynamic.NewMessage(msgDescriptor)
	err = jsonpb.Unmarshal(r.Body, msg)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to unmarshal into json: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg))
		return
	}

	reqBytes, err := proto.Marshal(msg)
	if err != nil {
		errMsg := fmt.Sprintf("Unable to marshal message: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(errMsg))
		return
	}

	// If qs was specified, encode the proto bytes and append to url
	if qs != "" {
		b64bytes := base64.URLEncoding.EncodeToString(reqBytes)
		urlstr := r.URL.String() + "?" + qs + "=" + b64bytes
		newurl, err := url.Parse(urlstr)
		if err != nil {
			errMsg := fmt.Sprintf("Error parsing url string: %v", urlstr)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(errMsg))
			return
		}
		r.URL = newurl
		r.ContentLength = 0
	} else {
		buffer := bytes.NewBuffer(reqBytes)
		r.Body = ioutil.NopCloser(buffer)
		r.ContentLength = int64(buffer.Len())
	}

	s.Proxy.ServeHTTP(w, r)
}

func (s *Server) Run() {
	http.HandleFunc("/", s.proxyRequest)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(s.Port)), nil))
}
