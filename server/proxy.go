package server

import (
	"bytes"
	"encoding/base64"
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
	Proxy *httputil.ReverseProxy
	Port  uint16
}

type Config struct {
	ProtoPath string
	Port      uint16
}

func New(cfg Config) *Server {
	var reqMsg, respMsg, qs string
	fd, err := fileDescriptorFromProto(cfg.ProtoPath)
	if err != nil {
		log.Printf("Error parsing protofile: %v", err)
		return nil
	}
	director := func(r *http.Request) {
		reqMsg, respMsg, qs, err = parseMessageTypes(r)
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

		// If qs was specified, encode the proto bytes and append to url
		if qs != "" {
			b64bytes := base64.URLEncoding.EncodeToString(reqBytes)
			urlstr := r.URL.String() + "?" + qs + "=" + b64bytes
			newurl, err := url.Parse(urlstr)
			if err != nil {
				log.Printf("Error parsing url string: %v", urlstr)
				return
			}
			r.URL = newurl
			r.ContentLength = 0
		} else {
			buffer := bytes.NewBuffer(reqBytes)
			r.Body = ioutil.NopCloser(buffer)
			r.ContentLength = int64(buffer.Len())
		}
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

	return &Server{
		Proxy: proxy,
		Port:  cfg.Port,
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

func (s *Server) Run() {
	http.HandleFunc("/", s.Proxy.ServeHTTP)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(s.Port)), nil))
}
