package main

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RussellLuo/timingwheel"
	"github.com/pires/go-proxyproto"
	"github.com/rs/zerolog/log"
)

type Destination struct {
	addr     *net.TCPAddr
	sessions atomic.Int64
}

type TcpReverseProxyGcScheduler struct {
}

func (t TcpReverseProxyGcScheduler) Next(now time.Time) time.Time {
	return now.Add(10 * time.Second)
}

type TcpReverseProxy struct {
	listenAddr       string
	destinations     map[string]*Destination
	destinationsLock sync.RWMutex
	currentLatest    string
	timingWheel      *timingwheel.TimingWheel
}

func NewTcpReverseProxy(addr string) *TcpReverseProxy {
	tw := timingwheel.NewTimingWheel(1*time.Second, 60)

	return &TcpReverseProxy{
		listenAddr:   addr,
		destinations: make(map[string]*Destination),
		timingWheel:  tw,
	}
}

func (trp *TcpReverseProxy) RenewDestination(name, addr string, cleanup func()) error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}

	trp.destinationsLock.Lock()
	latestName := trp.currentLatest
	trp.currentLatest = name
	trp.destinations[name] = &Destination{
		addr: tcpAddr,
	}
	trp.destinationsLock.Unlock()

	go func() {
		cancel := atomic.Pointer[func()]{}

		tm := trp.timingWheel.ScheduleFunc(TcpReverseProxyGcScheduler{}, func() {
			trp.destinationsLock.RLock()
			v, ok := trp.destinations[latestName]
			trp.destinationsLock.RUnlock()
			if !ok && cancel.Load() != nil {
				(*cancel.Load())()
				return
			}

			if v.sessions.Load() == 0 {
				trp.destinationsLock.Lock()
				delete(trp.destinations, latestName)
				trp.destinationsLock.Unlock()
				if cancel.Load() != nil {
					(*cancel.Load())()
				}
			}
		})

		cl := func() {
			log.Info().Str("name", name).Msg("triggered cleanup")
			tm.Stop()
			cleanup()
		}
		cancel.Store(&cl)
	}()

	return nil
}

func (trp *TcpReverseProxy) RemoveDestination(name string) {
	trp.destinationsLock.Lock()
	delete(trp.destinations, name)
	trp.destinationsLock.Unlock()
}

func (trp *TcpReverseProxy) Start(ctx context.Context) error {
	l, err := net.Listen("tcp", trp.listenAddr)
	if err != nil {
		return err
	}

	context.AfterFunc(ctx, func() {
		l.Close()
	})

	listenIP, err := net.ResolveTCPAddr("tcp", trp.listenAddr)
	if err != nil {
		return err
	}

	pl := proxyproto.Listener{
		Listener:          l,
		ReadHeaderTimeout: 5 * time.Second,
	}

	failedCount := 0
loop:
	for {
		log.Debug().Msg("waiting for connection")
		conn, err := pl.Accept()
		if err != nil || conn == nil {
			log.Error().Err(err).Msg("failed to accept connection")
			failedCount++
			if failedCount >= 5 {
				return err
			}
			continue loop
		}
		log.Debug().Str("address", conn.RemoteAddr().String()).Msg("accepted connection")
		failedCount = 0

		context.AfterFunc(ctx, func() {
			if conn != nil {
				conn.Close()
			}
		})

		go func() {
			if conn == nil {
				log.Error().Msg("connection is nil")
				return
			}
			remoteIpValue := conn.RemoteAddr().String()
			remoteIp, err := net.ResolveTCPAddr("tcp", remoteIpValue)
			if err != nil {
				log.Error().Err(err).Str("remote_ip", remoteIpValue).Msg("failed to resolve remote ip")
				return
			}

			trp.destinationsLock.RLock()
			latestName := trp.currentLatest
			dest := trp.destinations[latestName]
			trp.destinationsLock.RUnlock()
			if dest == nil {
				conn.Close()
				log.Error().Str("remote_ip", remoteIpValue).Str("latest_name", latestName).Msg("no destination found")
				return
			}

			dest.sessions.Add(1)
			defer dest.sessions.Add(-1)

			destinationConn, err := net.DialTCP("tcp", nil, dest.addr)
			if err != nil {
				conn.Close()
				log.Error().Err(err).Str("remote_ip", remoteIpValue).Str("latest_name", latestName).Str("destination", dest.addr.String()).Msg("failed to dial remote")
				return
			}

			context.AfterFunc(ctx, func() {
				destinationConn.Close()
			})

			protocol := proxyproto.TCPv6
			if remoteIp.IP.To4() != nil {
				protocol = proxyproto.TCPv4
			}

			header := &proxyproto.Header{
				Version:           2,
				Command:           proxyproto.PROXY,
				TransportProtocol: protocol,
				SourceAddr: &net.TCPAddr{
					IP:   remoteIp.IP,
					Port: remoteIp.Port,
				},
				DestinationAddr: &net.TCPAddr{
					IP:   listenIP.IP,
					Port: listenIP.Port,
				},
			}

			if _, err := header.WriteTo(destinationConn); err != nil {
				log.Error().Err(err).Str("remote_ip", remoteIpValue).Str("latest_name", latestName).Str("destination", dest.addr.String()).Msg("failed to write header")
				return
			}

			go func() {
				_, err := io.Copy(destinationConn, conn)
				destinationConn.Close()
				conn.Close()
				if err != nil {
					if errors.Is(err, net.ErrClosed) {
						log.Debug().Err(err).Str("remote_ip", remoteIpValue).Str("latest_name", latestName).Str("destination", dest.addr.String()).Msg("failed to copy to destination")
					} else {
						log.Error().Err(err).Str("remote_ip", remoteIpValue).Str("latest_name", latestName).Str("destination", dest.addr.String()).Msg("failed to copy to destination")
					}
				}
			}()

			go func() {
				_, err = io.Copy(conn, destinationConn)
				destinationConn.Close()
				conn.Close()
				if err != nil {
					if errors.Is(err, net.ErrClosed) {
						log.Debug().Err(err).Str("remote_ip", remoteIpValue).Str("latest_name", latestName).Str("destination", dest.addr.String()).Msg("failed to copy to remote")
					} else {
						log.Error().Err(err).Str("remote_ip", remoteIpValue).Str("latest_name", latestName).Str("destination", dest.addr.String()).Msg("failed to copy to remote")
					}
				}
			}()
		}()
	}
}
