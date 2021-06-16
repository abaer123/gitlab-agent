package grpctool_test

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/prototool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/matcher"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/mock_rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/testing/testhelpers"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	requestBodyData  = "jkasdbfkadsbfkadbfkjasbfkasbdf"
	responseBodyData = "nlnflkwqnflkasdnflnasdlfnasldnflnl"
)

func TestInboundGrpcToOutboundHttpStream_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	server := mock_rpc.NewMockInboundGrpcToOutboundHttpStream(ctrl)
	ctx := mock_modserver.IncomingCtx(context.Background(), t, testhelpers.AgentkToken)
	server.EXPECT().
		Context().
		Return(ctx).
		MinTimes(1)
	mockApi := mock_modserver.NewMockAPI(ctrl)
	sendHeader := &grpctool.HttpRequest_Header{
		Request: &prototool.HttpRequest{
			Method: "BOOM",
			Header: map[string]*prototool.Values{
				"cxv": {
					Value: []string{"xx"},
				},
			},
			UrlPath: "/asd/asd/asd",
			Query: map[string]*prototool.Values{
				"adasd": {
					Value: []string{"a"},
				},
			},
		},
		Extra: &anypb.Any{
			TypeUrl: "sadfasdfasdfads",
			Value:   []byte{1, 2, 3},
		},
	}
	gomock.InOrder(mockRecvStream(server, true,
		&grpctool.HttpRequest{
			Message: &grpctool.HttpRequest_Header_{
				Header: sendHeader,
			},
		},
		&grpctool.HttpRequest{
			Message: &grpctool.HttpRequest_Data_{
				Data: &grpctool.HttpRequest_Data{
					Data: []byte(requestBodyData[:1]),
				},
			},
		},
		&grpctool.HttpRequest{
			Message: &grpctool.HttpRequest_Data_{
				Data: &grpctool.HttpRequest_Data{
					Data: []byte(requestBodyData[1:]),
				},
			},
		},
		&grpctool.HttpRequest{
			Message: &grpctool.HttpRequest_Trailer_{
				Trailer: &grpctool.HttpRequest_Trailer{},
			},
		},
	)...)
	gomock.InOrder(mockSendStream(t, server,
		&grpctool.HttpResponse{
			Message: &grpctool.HttpResponse_Header_{
				Header: &grpctool.HttpResponse_Header{
					Response: &prototool.HttpResponse{
						StatusCode: http.StatusOK,
						Status:     "OK!",
						Header: map[string]*prototool.Values{
							"x1": {
								Value: []string{"a1", "a2"},
							},
						},
					},
				},
			},
		},
		&grpctool.HttpResponse{
			Message: &grpctool.HttpResponse_Data_{
				Data: &grpctool.HttpResponse_Data{
					Data: []byte(responseBodyData),
				},
			},
		},
		&grpctool.HttpResponse{
			Message: &grpctool.HttpResponse_Trailer_{
				Trailer: &grpctool.HttpResponse_Trailer{},
			},
		},
	)...)
	p := grpctool.NewInboundGrpcToOutboundHttp(mockApi, func(ctx context.Context, header *grpctool.HttpRequest_Header, body io.Reader) (*http.Response, error) {
		assert.Empty(t, cmp.Diff(header, sendHeader, protocmp.Transform()))
		data, err := ioutil.ReadAll(body)
		if !assert.NoError(t, err) {
			return nil, err
		}
		assert.Equal(t, requestBodyData, string(data))
		return &http.Response{
			Status:     "OK!",
			StatusCode: http.StatusOK,
			Header: http.Header{
				"x1": []string{"a1", "a2"},
			},
			Body: io.NopCloser(strings.NewReader(responseBodyData)),
		}, nil
	})
	err := p.Pipe(server)
	require.NoError(t, err)
}

func mockRecvStream(server *mock_rpc.MockInboundGrpcToOutboundHttpStream, eof bool, msgs ...proto.Message) []*gomock.Call {
	res := make([]*gomock.Call, 0, len(msgs)+1)
	for _, msg := range msgs {
		call := server.EXPECT().
			RecvMsg(gomock.Any()).
			Do(testhelpers.RecvMsg(msg))
		res = append(res, call)
	}
	if eof {
		call := server.EXPECT().
			RecvMsg(gomock.Any()).
			Return(io.EOF)
		res = append(res, call)
	}
	return res
}

func mockSendStream(t *testing.T, server *mock_rpc.MockInboundGrpcToOutboundHttpStream, msgs ...*grpctool.HttpResponse) []*gomock.Call {
	res := make([]*gomock.Call, 0, len(msgs))
	for _, msg := range msgs {
		call := server.EXPECT().
			SendMsg(matcher.ProtoEq(t, msg))
		res = append(res, call)
	}
	return res
}
