// Package server contains all of the logic for the proxy server.
package server

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/camgraff/protoxy/log"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

// Server is the base type for our proxy.
type Server struct {
	Port            uint16
	FileDescriptors []*desc.FileDescriptor
}

// Config holds the configuration for our server.
type Config struct {
	FileDescriptors []*desc.FileDescriptor
	Port            uint16
}

// protoTypes are used to determine the message types used to convert data in the request and response bodies.
type protoTypes struct {
	requestMessage   string
	responseMessages []string
	queryStringParam string
}

// New returns a new proxy server instance
func New(cfg Config) *Server {
	return &Server{
		Port:            cfg.Port,
		FileDescriptors: cfg.FileDescriptors,
	}
}

func parseMessageTypes(r *http.Request) (ptypes protoTypes, err error) {
	ctype := r.Header.Get("Content-Type")
	_, params, err := mime.ParseMediaType(ctype)
	if err != nil {
		return ptypes, err
	}
	// respmsg can contain multiple response types
	dstMsgs := strings.Split(params["respmsg"], ",")
	return protoTypes{
		requestMessage:   params["reqmsg"],
		responseMessages: dstMsgs,
		queryStringParam: params["qs"],
	}, nil
}

func writeErrorResponse(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Protoxy was unable to successfully proxy the request. See logs for details."))
}

func jsonBodyToProto(r *http.Request, msgDescriptor *desc.MessageDescriptor, qsParam string) error {
	msg := dynamic.NewMessage(msgDescriptor)
	err := jsonpb.Unmarshal(r.Body, msg)
	if err != nil {
		log.Log.WithError(err).Error("unable to unmarshal into json")
		return fmt.Errorf("Unable to unmarshal into json: %v", err)
	}

	reqBytes, err := proto.Marshal(msg)
	if err != nil {
		log.Log.WithError(err).Error("unable to marshal message")
		return fmt.Errorf("Unable to marshal message: %v", err)
	}

	// If qs was specified, encode the proto bytes and append to url
	if qsParam != "" {
		b64bytes := base64.URLEncoding.EncodeToString(reqBytes)
		urlstr := r.URL.String() + "?" + qsParam + "=" + b64bytes
		newurl, err := url.Parse(urlstr)
		if err != nil {
			log.Log.WithError(err).Error("error parsing url string")
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
	if reqMsg != "" && reqMsgDesc == nil {
		errMsg += fmt.Sprintf("Failed to find message descriptor for '%v'. ", reqMsg)
	}
	if len(respMsgs) > 0 && len(respMsgDescs) == 0 {
		errMsg += fmt.Sprintf("Failed to find any message descriptors for '%v'.", respMsgs)
	}
	if errMsg != "" {
		log.Log.WithField("err", errMsg).Error("failed to find message descriptors")
		return nil, nil, errors.New(errMsg)
	}
	return reqMsgDesc, respMsgDescs, nil
}

func (s *Server) proxyRequest(w http.ResponseWriter, r *http.Request) {
	msgTypes, err := parseMessageTypes(r)
	if err != nil {
		log.Log.WithError(err).Error("error parsing message types")
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	reqMsgDesc, respMsgDescs, err := s.findMessageDescriptors(msgTypes.requestMessage, msgTypes.responseMessages)
	if err != nil {
		log.Log.WithError(err).Error("error finding message descriptors")
		writeErrorResponse(w, http.StatusBadRequest)
		return
	}

	if reqMsgDesc != nil {
		if err = jsonBodyToProto(r, reqMsgDesc, msgTypes.queryStringParam); err != nil {
			log.Log.WithError(err).Error("error converting JSON body to proto")
			writeErrorResponse(w, http.StatusBadRequest)
			return
		}
	}

	// Override content-type to remove params
	r.Header.Set("Content-Type", "application/x-protobuf")

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

		marshaler := jsonpb.Marshaler{
			EmitDefaults: true,
		}
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
		log.Log.WithError(err).Error("unable to proxy response from server")
		writeErrorResponse(w, http.StatusBadRequest)
	}

	proxy := &httputil.ReverseProxy{
		Director:       func(*http.Request) {},
		ModifyResponse: modifyResp,
		ErrorHandler:   errorHandler,
	}

	proxy.ServeHTTP(w, r)
}

// Run starts the proxy server.
func (s *Server) Run() {
	http.HandleFunc("/", s.proxyRequest)
	log.Log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(s.Port)), nil))
}
