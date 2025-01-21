package listener

import (
	"net"

	"github.com/pires/go-proxyproto"
)

func New(addr string) (net.Listener, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	proxyListener := proxyproto.Listener{
		Listener: lis,
	}

	return &proxyListener, nil
}
