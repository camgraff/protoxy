package server

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/camgraff/protoxy/internal/testprotos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestProxy(t *testing.T) {
	tt := []struct {
		name               string
		protoPath          string
		reqBody            string
		reqHeader          string
		expectedRespBody   string
		expectedStatusCode int
	}{
		{
			name:               "happy",
			protoPath:          "../internal/testprotos/hello.proto",
			reqBody:            `{"text":"some text","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg=testprotos.Resp;`,
			expectedRespBody:   `{"text":"This is a response"}`,
			expectedStatusCode: 200,
		},
		{
			name:               "bad request message type",
			protoPath:          "../internal/testprotos/hello.proto",
			reqBody:            `{"text":"some text","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.DoesntExist; respmsg=testprotos.Resp;`,
			expectedRespBody:   "",
			expectedStatusCode: 400,
		},
		{
			name:               "bad response message type",
			protoPath:          "../internal/testprotos/hello.proto",
			reqBody:            `{"text":"some text","number":123,"list":["this","is","a","list"]}`,
			reqHeader:          `application/x-protobuf; reqmsg=testprotos.Req; respmsg=testprotos.DoesntExist;`,
			expectedRespBody:   "",
			expectedStatusCode: 400,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// Backend expects a request of prototests.Req and will respond with prototest.Resp
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				pb := &testprotos.Req{}
				body, err := ioutil.ReadAll(r.Body)
				require.NoError(t, err)
				proto.Unmarshal(body, pb)
				respPB := &testprotos.Resp{Text: "This is a response"}
				resp, err := proto.Marshal(respPB)
				require.NoError(t, err)
				w.Write(resp)
			}))

			req := httptest.NewRequest("GET", backend.URL, strings.NewReader(tc.reqBody))
			req.Header.Add("Content-Type", tc.reqHeader)
			respRecorder := httptest.NewRecorder()
			srv := New(Config{tc.protoPath, 7777})
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
