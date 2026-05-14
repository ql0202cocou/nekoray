package main

import (
	"context"
	"net"
	"net/http"

	"github.com/matsuridayo/libneko/neko_common"
	"github.com/matsuridayo/libneko/neko_log"
	box "github.com/sagernet/sing-box"
)

var instance *box.Box
var instance_cancel context.CancelFunc

func setupCore() {
	//
	neko_log.SetupLog(50*1024, "./neko.log")
	//
	neko_common.GetCurrentInstance = func() interface{} {
		mu.Lock()
		i := instance
		mu.Unlock()
		return i
	}
	neko_common.DialContext = func(ctx context.Context, specifiedInstance interface{}, network, addr string) (net.Conn, error) {
		if i, ok := specifiedInstance.(*box.Box); ok {
			return nekoDialContext(ctx, i, network, addr)
		}
		mu.Lock()
		i := instance
		mu.Unlock()
		if i != nil {
			return nekoDialContext(ctx, i, network, addr)
		}
		return neko_common.DialContextSystem(ctx, network, addr)
	}
	neko_common.CreateProxyHttpClient = func(specifiedInstance interface{}) *http.Client {
		if i, ok := specifiedInstance.(*box.Box); ok {
			return nekoCreateProxyHttpClient(i)
		}
		mu.Lock()
		i := instance
		mu.Unlock()
		if i == nil {
			return &http.Client{}
		}
		return nekoCreateProxyHttpClient(i)
	}
}
