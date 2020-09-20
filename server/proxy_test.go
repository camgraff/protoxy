package server

import (
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/camgraff/protoxy/internal/moreprotos"
	"github.com/camgraff/protoxy/internal/testprotos"
	"github.com/camgraff/protoxy/protoparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestProxy(t *testing.T) {
	// testProtoBackend expects a request of testprotos.Req and will respond with testprotos.Resp
	testProtoBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pb := &testprotos.Req{}
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		err = proto.Unmarshal(body, pb)
		require.NoError(t, err)

		respPB := &testprotos.Resp{Text: "This is a response"}
		resp, err := proto.Marshal(respPB)
		require.NoError(t, err)
		w.Write(resp)
	}))
	defer testProtoBackend.Close()

	// qsBackend is like testProtoBackend except it reads proto messages from the querystring
	qsBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pb := &testprotos.Req{}
		params, err := url.ParseQuery(r.URL.RawQuery)
		require.NoError(t, err)
		protoBytes, err := base64.URLEncoding.DecodeString(params["proto_body"][0])
		require.NoError(t, err)
		err = proto.Unmarshal(protoBytes, pb)
		require.NoError(t, err)

		respPB := &testprotos.Resp{Text: "This is a response"}
		resp, err := proto.Marshal(respPB)
		require.NoError(t, err)
		w.Write(resp)
	}))
	defer qsBackend.Close()

	// moreProtosBackend accepts a moreprotos.Req and returns a testprotos.Resp
	moreProtosBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pb := &moreprotos.Req{}
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		err = proto.Unmarshal(body, pb)
		require.NoError(t, err)

		respPB := &testprotos.Resp{Text: "This is a response"}
		resp, err := proto.Marshal(respPB)
		require.NoError(t, err)
		w.Write(resp)
	}))
	defer qsBackend.Close()

	tt := []struct {
		name               string
		importPaths        []string
		protoFiles         []string
		backend            *httptest.Server
		reqBody            string
		reqHeader          string
		expectedRespBody   string
		expectedStatusCode int
	}{
		{
			name:               "happy",
			importPaths:        []string{"../internal/testprotos"},
			protoFiles:         []string{"hello.proto"},
			reqBody:            `{"text":"some text","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg=testprotos.Resp;`,
			expectedRespBody:   `{"text":"This is a response"}`,
			expectedStatusCode: http.StatusOK,
			backend:            testProtoBackend,
		},
		{
			name:               "happy with querystring",
			importPaths:        []string{"../internal/testprotos"},
			protoFiles:         []string{"hello.proto"},
			reqBody:            `{"text":"some text","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg=testprotos.Resp; qs=proto_body`,
			expectedRespBody:   `{"text":"This is a response"}`,
			expectedStatusCode: http.StatusOK,
			backend:            qsBackend,
		},
		{
			name:               "no message types specified",
			importPaths:        []string{"../internal/testprotos"},
			protoFiles:         []string{"hello.proto"},
			reqHeader:          "application/x-protobuf",
			expectedStatusCode: http.StatusBadRequest,
			backend:            testProtoBackend,
		},
		{
			name:               "bad request message type",
			importPaths:        []string{"../internal/testprotos"},
			protoFiles:         []string{"hello.proto"},
			reqBody:            `{"text":"some text","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.DoesntExist; respmsg=testprotos.Resp;`,
			expectedStatusCode: http.StatusBadRequest,
			backend:            testProtoBackend,
		},
		{
			name:               "bad response message type",
			importPaths:        []string{"../internal/testprotos"},
			protoFiles:         []string{"hello.proto"},
			reqBody:            `{"text":"some text","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg=testprotos.DoesntExist;`,
			expectedStatusCode: http.StatusBadRequest,
			backend:            testProtoBackend,
		},
		{
			name:               "bad content type header",
			importPaths:        []string{"../internal/testprotos"},
			protoFiles:         []string{"hello.proto"},
			reqHeader:          "invalid",
			expectedStatusCode: http.StatusBadRequest,
			backend:            testProtoBackend,
		},
		{
			name:               "bad request body",
			importPaths:        []string{"../internal/testprotos"},
			protoFiles:         []string{"hello.proto"},
			reqBody:            `{"bad key":"bad value"}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg=testprotos.Resp;`,
			expectedStatusCode: http.StatusBadRequest,
			backend:            testProtoBackend,
		},

		{
			name:        "multiple import paths",
			importPaths: []string{"../internal/testprotos", "../internal/moreprotos"},
			protoFiles:  []string{"hello.proto", "moreprotos.proto"},
			reqBody: `{"subReq":
					  	{"text":"some text","number":123,"list":["this","is","a","list"]},
						"num": 22
					  }`,
			expectedRespBody:   `{"text":"This is a response"}`,
			reqHeader:          `application/x-protobuf; reqmsg=moreprotos.Req; respmsg=testprotos.Resp;`,
			expectedStatusCode: http.StatusOK,
			backend:            moreProtosBackend,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Create file descriptors
			fds, err := protoparser.FileDescriptorsFromPaths(tc.importPaths, tc.protoFiles)
			require.NoError(t, err)

			// Make the request
			req := httptest.NewRequest("GET", tc.backend.URL, strings.NewReader(tc.reqBody))
			req.Header.Add("Content-Type", tc.reqHeader)
			respRecorder := httptest.NewRecorder()
			srv := New(Config{fds, 7777})
			srv.proxyRequest(respRecorder, req)

			// Verify response
			assert.Equal(t, tc.expectedStatusCode, respRecorder.Code)
			if tc.expectedStatusCode < 300 {
				resp, err := ioutil.ReadAll(respRecorder.Body)
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedRespBody, string(resp))
			}
		})
	}

}
