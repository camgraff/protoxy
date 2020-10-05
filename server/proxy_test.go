package server

import (
	"encoding/base64"
	"io/ioutil"
	"mime"
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

func newBackend(t *testing.T, req proto.Message, resp proto.Message, querystring bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertHeaderParamsHaveBeenStripped(t, r)
		var body []byte
		var err error
		if querystring {
			params, err := url.ParseQuery(r.URL.RawQuery)
			require.NoError(t, err)
			body, err = base64.URLEncoding.DecodeString(params["proto_body"][0])
			require.NoError(t, err)

		} else {
			body, err = ioutil.ReadAll(r.Body)
			require.NoError(t, err)
		}
		err = proto.Unmarshal(body, req)
		require.NoError(t, err)
		resp, err := proto.Marshal(resp)
		require.NoError(t, err)
		w.Write(resp)
	}))
}

func newMultRespBackend(t *testing.T, resp1 proto.Message, resp2 proto.Message) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertHeaderParamsHaveBeenStripped(t, r)
		var body []byte
		var err error
		body, err = ioutil.ReadAll(r.Body)
		require.NoError(t, err)

		var req testprotos.Req
		err = proto.Unmarshal(body, &req)
		require.NoError(t, err)

		var resp []byte
		if req.Text == "want resp2" {
			resp, err = proto.Marshal(resp2)
		} else {
			resp, err = proto.Marshal(resp1)
		}
		require.NoError(t, err)
		w.Write(resp)
	}))
}

func assertHeaderParamsHaveBeenStripped(t *testing.T, r *http.Request) {
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	require.NoError(t, err)

	_, ok := params["reqmsg"]
	assert.False(t, ok)
	_, ok = params["respmsg"]
	assert.False(t, ok)
	_, ok = params["qs"]
	assert.False(t, ok)
}

func TestProxy(t *testing.T) {
	resp := &testprotos.Resp{Text: "This is a response"}
	// testProtoBackend expects a request of testprotos.Req and will respond with testprotos.Resp
	testProtoBackend := newBackend(t, &testprotos.Req{}, resp, false)
	defer testProtoBackend.Close()

	// qsBackend is like testProtoBackend except it reads proto messages from the querystring
	qsBackend := newBackend(t, &testprotos.Req{}, resp, true)
	defer qsBackend.Close()

	// moreProtosBackend accepts a moreprotos.Req and returns a testprotos.Resp
	moreProtosBackend := newBackend(t, &moreprotos.Req{}, resp, false)
	defer moreProtosBackend.Close()

	resp2 := &testprotos.Resp2{Number: 44}
	// multRespBackend sends a different response type depending on the input
	multRespBackend := newMultRespBackend(t, resp, resp2)
	defer multRespBackend.Close()

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
			name:               "multiple response types 1",
			importPaths:        []string{"../internal/testprotos"},
			protoFiles:         []string{"hello.proto"},
			backend:            multRespBackend,
			reqBody:            `{"text":"want resp1","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg="testprotos.Resp,testprotos.Resp2";`,
			expectedRespBody:   `{"text":"This is a response"}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "multiple response types 2",
			importPaths:        []string{"../internal/testprotos"},
			protoFiles:         []string{"hello.proto"},
			backend:            multRespBackend,
			reqBody:            `{"text":"want resp2","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg="testprotos.Resp,testprotos.Resp2";`,
			expectedRespBody:   `{"number":44}`,
			expectedStatusCode: http.StatusOK,
		},
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
