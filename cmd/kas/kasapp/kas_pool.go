package kasapp

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"google.golang.org/grpc"
)

type ClientConnInterface interface {
	grpc.ClientConnInterface
	io.Closer
}

var (
	_ KasPool = &defaultKasPool{}
)

type KasPool interface {
	Dial(ctx context.Context, target string) (ClientConnInterface, error)
}

// defaultKasPool is quite dumb at the moment. It needs to cache connections and close them when unused for some time.
type defaultKasPool struct {
	dialOpts []grpc.DialOption
}

func (p *defaultKasPool) Dial(ctx context.Context, target string) (ClientConnInterface, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "tcp":
		target = u.Host
	//case "tls":
	// TODO support TLS
	default:
		return nil, fmt.Errorf("unsupported kas URL scheme: %s", u.Scheme)
	}
	return grpc.DialContext(ctx, target, p.dialOpts...)
}
