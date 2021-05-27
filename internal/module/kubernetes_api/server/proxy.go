package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/gitlab"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/kubernetes_api/rpc"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/modserver"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/module/usage_metrics"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/grpctool"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/httpz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/logz"
	"gitlab.com/gitlab-org/cluster-integration/gitlab-agent/v14/internal/tool/prototool"
	"gitlab.com/gitlab-org/labkit/correlation"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	defaultMaxRequestDuration = 15 * time.Second
	shutdownTimeout           = defaultMaxRequestDuration
	readTimeout               = 1 * time.Second
	writeTimeout              = defaultMaxRequestDuration
	idleTimeout               = 1 * time.Minute
	maxDataChunkSize          = 32 * 1024

	authorizationHeader             = "Authorization"
	hostHeader                      = "Host"
	authorizationHeaderBearerPrefix = "Bearer " // must end with a space
	jobInfoApiPath                  = "/api/v4/job/allowed_agents"
	tokenSeparator                  = ":"
	tokenTypeCi                     = "ci"

	headerFieldNumber  protoreflect.FieldNumber = 1
	dataFieldNumber    protoreflect.FieldNumber = 2
	trailerFieldNumber protoreflect.FieldNumber = 3
)

var (
	// See https://httpwg.org/http-core/draft-ietf-httpbis-semantics-latest.html#field.connection
	// See https://tools.ietf.org/html/rfc2616#section-13.5.1
	// See https://github.com/golang/go/blob/81ea89adf38b90c3c3a8c4eed9e6c093a8634d59/src/net/http/httputil/reverseproxy.go#L169-L184
	hopHeaders = []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",      // canonicalized version of "TE"
		"Trailer", // not Trailers as per rfc2616; See errata https://www.rfc-editor.org/errata_search.php?eid=4522
		"Transfer-Encoding",
		"Upgrade",
	}
)

type allowedAgent struct {
	Id            int64         `json:"id"`
	ConfigProject configProject `json:"config_project"`
}

type configProject struct {
	Id int64 `json:"id"`
}

type pipeline struct {
	Id int64 `json:"id"`
}

type project struct {
	Id int64 `json:"id"`
}

type job struct {
	Id int64 `json:"id"`
}

type user struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
}

type jobInfo struct {
	AllowedAgents []allowedAgent `json:"allowed_agents"`
	Job           job            `json:"job"`
	Pipeline      pipeline       `json:"pipeline"`
	Project       project        `json:"project"`
	User          user           `json:"user"`
}

type kubernetesApiProxy struct {
	log                 *zap.Logger
	api                 modserver.API
	kubernetesApiClient rpc.KubernetesApiClient
	gitLabClient        gitlab.ClientInterface
	streamVisitor       *grpctool.StreamVisitor
	requestCount        usage_metrics.Counter
	serverName          string
	// urlPathPrefix is guaranteed to end with / by defaulting.
	urlPathPrefix string
}

func (p *kubernetesApiProxy) Run(ctx context.Context, listener net.Listener) error {
	srv := &http.Server{
		Handler:      http.HandlerFunc(p.proxy),
		WriteTimeout: writeTimeout,
		ReadTimeout:  readTimeout,
		IdleTimeout:  idleTimeout,
	}
	return httpz.RunServer(ctx, srv, listener, shutdownTimeout)
}

func (p *kubernetesApiProxy) proxy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Server", p.serverName) // This might be overwritten by remote (e.g. if there is no error)

	correlationId := correlation.SafeRandomID()
	log := p.log.With(logz.CorrelationId(correlationId))

	agentId, jobToken, err := getAgentIdAndJobTokenFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		log.Debug("Unauthorized: header", zap.Error(err))
		return
	}
	log = log.With(logz.AgentId(agentId))
	ctx := correlation.ContextWithCorrelation(r.Context(), correlationId)

	jInfo, err := p.getJobInfo(ctx, jobToken)
	if err != nil {
		switch {
		case gitlab.IsUnauthorized(err):
			w.WriteHeader(http.StatusUnauthorized)
			log.Debug("Unauthorized: CI job token")
		case gitlab.IsForbidden(err):
			w.WriteHeader(http.StatusForbidden)
			log.Debug("Forbidden: CI job token")
		default:
			w.WriteHeader(http.StatusInternalServerError)
			p.api.HandleProcessingError(ctx, log, "Failed to get allowed agents for CI job token", err)
		}
		return
	}

	aa := findAllowedAgent(agentId, jInfo)
	if aa == nil {
		w.WriteHeader(http.StatusForbidden)
		log.Debug("Forbidden: agentId is not allowed")
		return
	}

	if !strings.HasPrefix(r.URL.Path, p.urlPathPrefix) {
		w.WriteHeader(http.StatusBadRequest)
		log.Debug("Bad request: URL does not start with expected prefix", logz.UrlPath(r.URL.Path), logz.UrlPathPrefix(p.urlPathPrefix))
		return
	}
	// urlPathPrefix is guaranteed to end with / by defaulting. That means / will be removed here.
	// Put it back by -1 on length.
	r.URL.Path = r.URL.Path[len(p.urlPathPrefix)-1:]

	p.requestCount.Inc() // Count only authenticated and authorized requests

	headerWritten, errF := p.pipeStreams(ctx, log, w, r, agentId)
	if errF != nil {
		if headerWritten {
			// HTTP status has been written already as part of the normal response flow.
			// But then something went wrong and an error happened. To let the client know that something isn't right
			// we have only one thing we can do - abruptly close the connection. To do that we panic with a special
			// error value that the "http" package provides. See its description.
			// If we try to write the status again here, http package would log a warning, which is not nice.
			panic(http.ErrAbortHandler)
		} else {
			errF(w) // nolint: errorlint
		}
	}
}

func (p *kubernetesApiProxy) pipeStreams(ctx context.Context, log *zap.Logger, w http.ResponseWriter, r *http.Request, agentId int64) (bool /* headerWritten */, errFunc) {
	g, ctx := errgroup.WithContext(ctx)
	md := metadata.Pairs(modserver.RoutingAgentIdMetadataKey, strconv.FormatInt(agentId, 10))
	mkClient, err := p.kubernetesApiClient.MakeRequest(metadata.NewOutgoingContext(ctx, md)) // must use context from errgroup
	if err != nil {
		p.api.HandleProcessingError(ctx, log, "Proxy failed to make outbound request", err)
		return false, p.http502
	}
	// Pipe client -> remote
	g.Go(func() error {
		errFuncRet := p.pipeClientToRemote(ctx, log, mkClient, r)
		if errFuncRet != nil {
			return errFuncRet
		}
		return nil
	})
	// Pipe remote -> client
	headerWritten := false
	g.Go(func() error {
		var errFuncRet errFunc
		headerWritten, errFuncRet = p.pipeRemoteToClient(ctx, log, w, mkClient)
		if errFuncRet != nil {
			return errFuncRet
		}
		return nil
	})
	err = g.Wait() // don't inline as headerWritten must be read after Wait() returned
	if err != nil {
		return headerWritten, err.(errFunc) // nolint: errorlint
	}
	return false, nil
}

func (p *kubernetesApiProxy) pipeRemoteToClient(ctx context.Context, log *zap.Logger, w http.ResponseWriter, mkClient rpc.KubernetesApi_MakeRequestClient) (bool /* headerWritten */, errFunc) {
	writeFailed := false
	headerWritten := false
	err := p.streamVisitor.Visit(mkClient,
		grpctool.WithCallback(headerFieldNumber, func(header *grpctool.HttpResponse_Header) error {
			httpH := header.Response.HttpHeader()
			httpz.RemoveConnectionHeaders(httpH)
			h := w.Header()
			for k, vals := range httpH {
				h[k] = vals
			}
			w.WriteHeader(int(header.Response.StatusCode))
			headerWritten = true
			return nil
		}),
		grpctool.WithCallback(dataFieldNumber, func(data *grpctool.HttpResponse_Data) error {
			_, err := w.Write(data.Data)
			if err != nil {
				writeFailed = true
			}
			return err
		}),
		grpctool.WithCallback(trailerFieldNumber, func(trailer *grpctool.HttpResponse_Trailer) error {
			return nil
		}),
	)
	if err != nil {
		if writeFailed {
			// there is likely a connection problem so the client will likely not receive this
			return headerWritten, p.http400
		}
		p.api.HandleProcessingError(ctx, log, "Proxy failed to read response body", err)
		return headerWritten, p.http502
	}
	return headerWritten, nil
}

func (p *kubernetesApiProxy) pipeClientToRemote(ctx context.Context, log *zap.Logger, mkClient rpc.KubernetesApi_MakeRequestClient, r *http.Request) errFunc {
	err := mkClient.Send(&grpctool.HttpRequest{
		Message: &grpctool.HttpRequest_Header_{
			Header: &grpctool.HttpRequest_Header{
				Request: &prototool.HttpRequest{
					Method:  r.Method,
					Header:  headerFromHttpRequestHeader(r.Header),
					UrlPath: r.URL.Path,
					Query:   prototool.UrlValuesToValuesMap(r.URL.Query()),
				},
			},
		},
	})
	if err != nil {
		_ = p.api.HandleSendError(log, "Proxy failed to send header", err)
		return p.http502
	}

	buffer := make([]byte, maxDataChunkSize)
	for {
		var n int
		n, err = r.Body.Read(buffer)
		if err != nil && !errors.Is(err, io.EOF) {
			p.api.HandleProcessingError(ctx, log, "Proxy failed to read request body", err)
			return p.http400 // there is likely a connection problem so the client will likely not receive this
		}
		if n > 0 { // handle n=0, err=io.EOF case
			sendErr := mkClient.Send(&grpctool.HttpRequest{
				Message: &grpctool.HttpRequest_Data_{
					Data: &grpctool.HttpRequest_Data{
						Data: buffer[:n],
					},
				},
			})
			if sendErr != nil {
				_ = p.api.HandleSendError(log, "Proxy failed to send data", sendErr)
				return p.http502
			}
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
	err = mkClient.Send(&grpctool.HttpRequest{
		Message: &grpctool.HttpRequest_Trailer_{
			Trailer: &grpctool.HttpRequest_Trailer{},
		},
	})
	if err != nil {
		_ = p.api.HandleSendError(log, "Proxy failed to send trailers", err)
		return p.http502
	}
	err = mkClient.CloseSend()
	if err != nil {
		_ = p.api.HandleSendError(log, "Proxy failed to send close frame", err)
		return p.http502
	}
	return nil
}

func (p *kubernetesApiProxy) getJobInfo(ctx context.Context, jobToken string) (*jobInfo, error) {
	ji := &jobInfo{}
	err := p.gitLabClient.Do(ctx,
		gitlab.WithPath(jobInfoApiPath),
		gitlab.WithJobToken(jobToken),
		gitlab.WithResponseHandler(gitlab.JsonResponseHandler(ji)),
	)
	if err != nil {
		return nil, err
	}
	return ji, nil
}

func findAllowedAgent(agentId int64, jInfo *jobInfo) *allowedAgent {
	for _, aa := range jInfo.AllowedAgents {
		if aa.Id == agentId {
			return &aa
		}
	}
	return nil
}

func getAgentIdAndJobTokenFromRequest(r *http.Request) (int64, string, error) {
	h := r.Header.Values(authorizationHeader)
	if len(h) == 0 {
		return 0, "", fmt.Errorf("%s header: expecting token", authorizationHeader)
	}
	if len(h) > 1 {
		return 0, "", fmt.Errorf("%s header: expecting a single header, got %d", authorizationHeader, len(h))
	}
	return getAgentIdAndJobTokenFromHeader(h[0])
}

func getAgentIdAndJobTokenFromHeader(header string) (int64, string, error) {
	if !strings.HasPrefix(header, authorizationHeaderBearerPrefix) {
		// "missing" space in message - it's in the authorizationHeaderBearerPrefix constant already
		return 0, "", fmt.Errorf("%s header: expecting %stoken", authorizationHeader, authorizationHeaderBearerPrefix)
	}
	tokenValue := header[len(authorizationHeaderBearerPrefix):]
	tokenValueParts := strings.SplitN(tokenValue, tokenSeparator, 2)
	if len(tokenValueParts) != 2 {
		return 0, "", fmt.Errorf("%s header: invalid value", authorizationHeader)
	}
	switch tokenValueParts[0] {
	case tokenTypeCi:
	default:
		return 0, "", fmt.Errorf("%s header: unknown token type", authorizationHeader)
	}
	agentIdAndToken := tokenValueParts[1]
	agentIdAndTokenParts := strings.SplitN(agentIdAndToken, tokenSeparator, 2)
	if len(agentIdAndTokenParts) != 2 {
		return 0, "", fmt.Errorf("%s header: invalid value", authorizationHeader)
	}
	agentId, err := strconv.ParseInt(agentIdAndTokenParts[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("%s header: failed to parse: %v", authorizationHeader, err)
	}
	token := agentIdAndTokenParts[1]
	if token == "" {
		return 0, "", fmt.Errorf("%s header: empty token", authorizationHeader)
	}
	return agentId, token, nil
}

func headerFromHttpRequestHeader(header http.Header) map[string]*prototool.Values {
	header = header.Clone()
	header.Del(hostHeader)          // Use the destination host name
	header.Del(authorizationHeader) // Remove Authorization header - we got the CI job token in it

	// Remove hop-by-hop headers
	// 1. Remove headers listed in the Connection header
	httpz.RemoveConnectionHeaders(header)
	// 2. Remove well-known headers
	for _, name := range hopHeaders {
		header.Del(name)
	}

	return prototool.HttpHeaderToValuesMap(header)
}

// See https://tools.ietf.org/html/rfc7231#section-6.5.1
func (p *kubernetesApiProxy) http400(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadRequest)
}

// See https://tools.ietf.org/html/rfc7231#section-6.6.3
func (p *kubernetesApiProxy) http502(w http.ResponseWriter) {
	w.WriteHeader(http.StatusBadGateway)
}

var (
	_ error = errFunc(nil)
)

// errFunc enhances type safety.
type errFunc func(http.ResponseWriter)

func (e errFunc) Error() string {
	return "errorFunc"
}
