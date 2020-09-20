package server

import (
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/camgraff/protoxy/internal/testprotos"
	"github.com/camgraff/protoxy/protoparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestProxy(t *testing.T) {
	const importPath = "../internal/testprotos/"
	const protoFile = "hello.proto"
	fds, err := protoparser.FileDescriptorsFromPaths([]string{importPath}, []string{protoFile})
	require.NoError(t, err)

	tt := []struct {
		name               string
		protoPath          string
		reqBody            string
		reqHeader          string
		expectedRespBody   string
		expectedStatusCode int
		hasQuerysting      bool
	}{
		{
			name:               "happy",
			reqBody:            `{"text":"some text","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg=testprotos.Resp;`,
			expectedRespBody:   `{"text":"This is a response"}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "happy with querystring",
			reqBody:            `{"text":"some text","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg=testprotos.Resp; qs=proto_body`,
			expectedRespBody:   `{"text":"This is a response"}`,
			expectedStatusCode: http.StatusOK,
			hasQuerysting:      true,
		},
		{
			name:               "no message types specified",
			reqHeader:          "application/x-protobuf",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "bad request message type",
			reqBody:            `{"text":"some text","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.DoesntExist; respmsg=testprotos.Resp;`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "bad response message type",
			reqBody:            `{"text":"some text","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg=testprotos.DoesntExist;`,
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "bad content type header",
			reqHeader:          "invalid",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "bad request body",
			reqBody:            `{"bad key":"bad value"}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg=testprotos.Resp;`,
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Backend expects a request of prototests.Req and will respond with prototest.Resp
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pb := &testprotos.Req{}
				if tc.hasQuerysting {
					params, err := url.ParseQuery(r.URL.RawQuery)
					require.NoError(t, err)
					protoBytes, err := base64.URLEncoding.DecodeString(params["proto_body"][0])
					require.NoError(t, err)
					err = proto.Unmarshal(protoBytes, pb)
					require.NoError(t, err)
				} else {
					body, err := ioutil.ReadAll(r.Body)
					require.NoError(t, err)
					err = proto.Unmarshal(body, pb)
					require.NoError(t, err)
				}
				respPB := &testprotos.Resp{Text: "This is a response"}
				resp, err := proto.Marshal(respPB)
				require.NoError(t, err)
				w.Write(resp)
			}))
			defer backend.Close()

			req := httptest.NewRequest("GET", backend.URL, strings.NewReader(tc.reqBody))
			req.Header.Add("Content-Type", tc.reqHeader)
			respRecorder := httptest.NewRecorder()
			srv := New(Config{fds, 7777})
			srv.proxyRequest(respRecorder, req)

			assert.Equal(t, tc.expectedStatusCode, respRecorder.Code)

			if tc.expectedStatusCode < 300 {
				resp, err := ioutil.ReadAll(respRecorder.Body)
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedRespBody, string(resp))
			}
		})
	}

}
