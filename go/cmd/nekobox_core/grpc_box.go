package main

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"grpc_server"
	"grpc_server/gen"

	"github.com/matsuridayo/libneko/neko_common"
	"github.com/matsuridayo/libneko/speedtest"
	box "github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/experimental/v2rayapi"
	"github.com/sagernet/sing-box/option"

	"log"
)

var statsService *v2rayapi.StatsService

// mu protects instance, instance_cancel, and statsService from concurrent access
var mu sync.Mutex

type server struct {
	grpc_server.BaseServer
}

func (s *server) Start(ctx context.Context, in *gen.LoadConfigReq) (out *gen.ErrorResp, _ error) {
	var err error

	defer func() {
		out = &gen.ErrorResp{}
		if err != nil {
			out.Error = err.Error()
			mu.Lock()
			instance = nil
			mu.Unlock()
		}
	}()

	if neko_common.Debug {
		// M2 fix: Don't log full config (may contain credentials)
		log.Printf("Start: config length=%d", len(in.CoreConfig))
	}

	mu.Lock()
	if instance != nil {
		mu.Unlock()
		err = errors.New("instance already started")
		return
	}
	mu.Unlock()

	newInstance, newCancel, createErr := createBox([]byte(in.CoreConfig))

	mu.Lock()
	instance = newInstance
	instance_cancel = newCancel
	err = createErr

	if instance != nil {
		// V2ray Service
		if in.StatsOutbounds != nil {
			statsService = v2rayapi.NewStatsService(option.V2RayStatsServiceOptions{
				Enabled:   true,
				Outbounds: in.StatsOutbounds,
			})
			instance.Router().AppendTracker(statsService)
		}
	}
	mu.Unlock()

	return
}

func (s *server) Stop(ctx context.Context, in *gen.EmptyReq) (out *gen.ErrorResp, _ error) {
	var err error

	defer func() {
		out = &gen.ErrorResp{}
		if err != nil {
			out.Error = err.Error()
		}
	}()

	mu.Lock()
	if instance == nil {
		mu.Unlock()
		return
	}

	cancel := instance_cancel
	inst := instance
	instance = nil
	instance_cancel = nil
	statsService = nil
	mu.Unlock()

	// Close outside lock to avoid deadlock if Close blocks
	cancel()
	inst.Close()

	return
}

func (s *server) Test(ctx context.Context, in *gen.TestReq) (out *gen.TestResp, _ error) {
	var err error
	out = &gen.TestResp{Ms: 0}

	defer func() {
		if err != nil {
			out.Error = err.Error()
		}
	}()

	if in.Mode == gen.TestMode_UrlTest {
		var i *box.Box
		var cancel context.CancelFunc
		if in.Config != nil {
			// Test instance
			i, cancel, err = createBox([]byte(in.Config.CoreConfig))
			if i != nil {
				defer i.Close()
				defer cancel()
			}
			if err != nil {
				return
			}
		} else {
			// Test running instance
			mu.Lock()
			i = instance
			mu.Unlock()
			if i == nil {
				return
			}
		}
		// Latency
		out.Ms, err = speedtest.UrlTest(nekoCreateProxyHttpClient(i), in.Url, in.Timeout, speedtest.UrlTestStandard_RTT)
	} else if in.Mode == gen.TestMode_TcpPing {
		out.Ms, err = speedtest.TcpPing(in.Address, in.Timeout)
	} else if in.Mode == gen.TestMode_FullTest {
		i, cancel, err := createBox([]byte(in.Config.CoreConfig))
		if i != nil {
			defer i.Close()
			defer cancel()
		}
		if err != nil {
			return
		}
		return grpc_server.DoFullTest(ctx, in, i)
	}

	return
}

func (s *server) QueryStats(ctx context.Context, in *gen.QueryStatsReq) (out *gen.QueryStatsResp, _ error) {
	out = &gen.QueryStatsResp{}

	mu.Lock()
	ss := statsService
	mu.Unlock()

	if ss != nil {
		pattern := fmt.Sprintf("outbound>>>%s>>>traffic>>>%s", in.Tag, in.Direct)
		response, err := ss.QueryStats(ctx, &v2rayapi.QueryStatsRequest{
			Patterns: []string{pattern},
		})
		if err == nil && len(response.Stat) > 0 {
			out.Traffic = response.Stat[0].Value
		}
	}

	return
}

func (s *server) ListConnections(ctx context.Context, in *gen.EmptyReq) (*gen.ListConnectionsResp, error) {
	out := &gen.ListConnectionsResp{
		// TODO upstream api
	}
	return out, nil
}
