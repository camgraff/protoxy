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
	"github.com/jhump/protoreflect/dynamic"
)

type Server struct {
	Port           uint16
	FileDescriptor *desc.FileDescriptor
}

type Config struct {
	FileDescriptor *desc.FileDescriptor
	Port           uint16
}

func New(cfg Config) *Server {
	return &Server{
		Port:           cfg.Port,
		FileDescriptor: cfg.FileDescriptor,
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

func writeErrorResponse(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	w.Write([]byte(err.Error()))
}

func jsonBodyToProto(r *http.Request, msgDescriptor *desc.MessageDescriptor) error {
	msg := dynamic.NewMessage(msgDescriptor)
	err := jsonpb.Unmarshal(r.Body, msg)
	if err != nil {
		return fmt.Errorf("Unable to unmarshal into json: %v", err)
	}

	reqBytes, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("Unable to marshal message: %v", err)
	}

	_, _, qs, err := parseMessageTypes(r)
	if err != nil {
		return fmt.Errorf("Error parsing content-type: %w", err)
	}

	// If qs was specified, encode the proto bytes and append to url
	if qs != "" {
		b64bytes := base64.URLEncoding.EncodeToString(reqBytes)
		urlstr := r.URL.String() + "?" + qs + "=" + b64bytes
		newurl, err := url.Parse(urlstr)
		if err != nil {
			return fmt.Errorf("Error parsing url string: %v", err)
		}
		r.URL = newurl
		r.ContentLength = 0
		return nil
	}

	buffer := bytes.NewBuffer(reqBytes)
	r.Body = ioutil.NopCloser(buffer)
	r.ContentLength = int64(buffer.Len())

	return nil
}

func (s *Server) proxyRequest(w http.ResponseWriter, r *http.Request) {
	reqMsg, respMsg, _, err := parseMessageTypes(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("Error parsing content-type: %w", err))
		return
	}

	msgDescriptor := s.FileDescriptor.FindMessage(reqMsg)
	if msgDescriptor == nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("Unable to find message: %v", err))
		return
	}

	err = jsonBodyToProto(r, msgDescriptor)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("Error converting JSON body to Protobuf: %w", err))
		return
	}

	modifyResp := func(r *http.Response) error {
		msgDescriptor := s.FileDescriptor.FindMessage(respMsg)
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

	proxy.ServeHTTP(w, r)
}

func (s *Server) Run() {
	http.HandleFunc("/", s.proxyRequest)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(s.Port)), nil))
}
