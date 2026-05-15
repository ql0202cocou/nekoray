package grpc_server

import (
	"context"
	"encoding/hex"
	"fmt"
	"grpc_server/gen"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/matsuridayo/libneko/neko_common"
	"github.com/matsuridayo/libneko/speedtest"
)

const (
	KiB = 1024
	MiB = 1024 * KiB
)

func getBetweenStr(str, start, end string) string {
	n := strings.Index(str, start)
	if n == -1 {
		n = 0
	}
	str = string([]byte(str)[n:])
	m := strings.Index(str, end)
	if m == -1 {
		m = len(str)
	}
	str = string([]byte(str)[:m])
	return str[len(start):]
}

func DoFullTest(ctx context.Context, in *gen.TestReq, instance interface{}) (out *gen.TestResp, _ error) {
	out = &gen.TestResp{}
	httpClient := neko_common.CreateProxyHttpClient(instance)

	// Latency
	var latency string
	if in.FullLatency {
		t, _ := speedtest.UrlTest(httpClient, in.Url, in.Timeout, speedtest.UrlTestStandard_RTT)
		out.Ms = t
		if t > 0 {
			latency = fmt.Sprint(t, "ms")
		} else {
			latency = "Error"
		}
	}

	// UDP Latency
	var udpLatency string
	if in.FullUdpLatency {
		udpCtx, cancel := context.WithTimeout(ctx, time.Second*3)
		result := make(chan string, 1)

		go func() {
			var startTime = time.Now()
			pc, err := neko_common.DialContext(udpCtx, instance, "udp", "8.8.8.8:53")
			if err == nil {
				defer pc.Close()
				// C5 fix: Set read deadline to ensure goroutine exits on timeout
				if err := pc.SetReadDeadline(time.Now().Add(time.Second * 3)); err != nil {
					log.Println("UDP SetReadDeadline error:", err)
				}
				dnsPacket, _ := hex.DecodeString("0000010000010000000000000377777706676f6f676c6503636f6d0000010001")
				_, err = pc.Write(dnsPacket)
				if err == nil {
					var buf [1400]byte
					_, err = pc.Read(buf[:])
				}
			}
			if err == nil {
				var endTime = time.Now()
				result <- fmt.Sprint(endTime.Sub(startTime).Abs().Milliseconds(), "ms")
			} else {
				log.Println("UDP Latency test error:", err)
				result <- "Error"
			}
			close(result)
		}()

		select {
		case <-udpCtx.Done():
			udpLatency = "Timeout"
		case r := <-result:
			udpLatency = r
		}
		cancel()
	}

	// 入口 IP
	var in_ip string
	if in.FullInOut {
		_in_ip, err := net.ResolveIPAddr("ip", in.InAddress)
		if err == nil {
			in_ip = _in_ip.String()
		} else {
			in_ip = err.Error()
		}
	}

	// 出口 IP
	var out_ip string
	if in.FullInOut {
		resp, err := httpClient.Get("https://www.cloudflare.com/cdn-cgi/trace")
		if err == nil {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				out_ip = "Error"
			} else {
				out_ip = getBetweenStr(string(b), "ip=", "\n")
			}
			resp.Body.Close()
		} else {
			out_ip = "Error"
		}
	}

	// 下载
	var speed string
	if in.FullSpeed {
		if in.FullSpeedTimeout <= 0 {
			in.FullSpeedTimeout = 30
		}

		dlCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(in.FullSpeedTimeout))
		result := make(chan string, 1)
		// C5 fix: Use channel to safely pass body closer between goroutines
		bodyChan := make(chan io.Closer, 1)

		go func() {
			req, err := http.NewRequestWithContext(dlCtx, "GET", in.FullSpeedUrl, nil)
			if err != nil {
				close(bodyChan)
				result <- "Error"
				close(result)
				return
			}
			resp, err := httpClient.Do(req)
			if err == nil && resp != nil && resp.Body != nil {
				// Send body closer to main goroutine before blocking on io.Copy
				bodyChan <- resp.Body
				defer resp.Body.Close()

				timeStart := time.Now()
				n, _ := io.Copy(io.Discard, resp.Body)
				timeEnd := time.Now()

				duration := math.Max(timeEnd.Sub(timeStart).Seconds(), 0.000001)
				resultSpeed := (float64(n) / duration) / MiB
				result <- fmt.Sprintf("%.2fMiB/s", resultSpeed)
			} else {
				close(bodyChan)
				result <- "Error"
			}
			close(result)
		}()

		select {
		case <-dlCtx.Done():
			speed = "Timeout"
		case s := <-result:
			speed = s
		}

		cancel()
		// Close body to interrupt io.Copy if it's still running
		if bc, ok := <-bodyChan; ok {
			bc.Close()
		}
	}

	fr := make([]string, 0)
	if latency != "" {
		fr = append(fr, fmt.Sprintf("Latency: %s", latency))
	}
	if udpLatency != "" {
		fr = append(fr, fmt.Sprintf("UDPLatency: %s", udpLatency))
	}
	if speed != "" {
		fr = append(fr, fmt.Sprintf("Speed: %s", speed))
	}
	if in_ip != "" {
		fr = append(fr, fmt.Sprintf("In: %s", in_ip))
	}
	if out_ip != "" {
		fr = append(fr, fmt.Sprintf("Out: %s", out_ip))
	}

	out.FullReport = strings.Join(fr, " / ")

	return
}
