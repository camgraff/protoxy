package server

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

type Server struct {
	Port            uint16
	FileDescriptors []*desc.FileDescriptor
}

type Config struct {
	FileDescriptors []*desc.FileDescriptor
	Port            uint16
}

func New(cfg Config) *Server {
	return &Server{
		Port:            cfg.Port,
		FileDescriptors: cfg.FileDescriptors,
	}
}

func parseMessageTypes(r *http.Request) (srcMsg string, dstMsgs []string, qs string, err error) {
	ctype := r.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(ctype)
	if err != nil {
		return "", nil, "", err
	}
	// respmsg can contain multiple response types
	dstMsgs = strings.Split(params["respmsg"], ",")
	return params["reqmsg"], dstMsgs, params["qs"], nil
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

func (s *Server) findMessageDescriptors(reqMsg string, respMsgs []string) (reqMsgDesc *desc.MessageDescriptor, respMsgDescs []*desc.MessageDescriptor, err error) {
	for _, fd := range s.FileDescriptors {
		if reqMsgDesc == nil {
			reqMsgDesc = fd.FindMessage(reqMsg)
		}
		for _, r := range respMsgs {
			possibleDesc := fd.FindMessage(r)
			if possibleDesc != nil {
				respMsgDescs = append(respMsgDescs, possibleDesc)
			}
		}
	}

	var errMsg string
	if reqMsgDesc == nil {
		errMsg += fmt.Sprintf("Failed to find message descriptor for '%v'. ", reqMsg)
	}
	if len(respMsgDescs) == 0 {
		errMsg += fmt.Sprintf("Failed to find any message descriptors for '%v'.", respMsgs)
	}
	if errMsg != "" {
		return nil, nil, errors.New(errMsg)
	}
	return reqMsgDesc, respMsgDescs, nil
}

func (s *Server) proxyRequest(w http.ResponseWriter, r *http.Request) {
	reqMsg, respMsgs, _, err := parseMessageTypes(r)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("Error parsing content-type: %w", err))
		return
	}

	reqMsgDesc, respMsgDescs, err := s.findMessageDescriptors(reqMsg, respMsgs)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err)
		return
	}

	err = jsonBodyToProto(r, reqMsgDesc)
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("Error converting JSON body to Protobuf: %w", err))
		return
	}

	modifyResp := func(r *http.Response) error {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("Failed to read response body: %v", err)
		}
		err = r.Body.Close()
		if err != nil {
			return fmt.Errorf("Error closing body: %v", err)
		}
		// Try all possible responses until something works
		var errs error
		var msg proto.Message
		for _, d := range respMsgDescs {
			msg = dynamic.NewMessage(d)
			err = proto.Unmarshal(body, msg)
			if err != nil {
				errs = fmt.Errorf("Unable to unmarshal into json: %v", err)
			} else {
				errs = nil
				break
			}
		}
		if errs != nil {
			return errs
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
		r.Header.Set("Content-Type", "application/json")
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
