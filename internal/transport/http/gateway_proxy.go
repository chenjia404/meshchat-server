package http

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// NewIPFSGatewayProxy 将请求反向代理到 Kubo 网关（例如 http://ipfs:8080）。
// 路由应挂在 /ipfs/，与网关路径一致，无需改写 Path。
func NewIPFSGatewayProxy(upstream *url.URL) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(upstream)
	proxy.ModifyResponse = func(resp *http.Response) error {
		rewriteIPFSGatewayLocation(resp, upstream)
		return nil
	}
	return proxy
}

func rewriteIPFSGatewayLocation(resp *http.Response, upstream *url.URL) {
	loc := resp.Header.Get("Location")
	if loc == "" {
		return
	}
	u, err := url.Parse(loc)
	if err != nil {
		return
	}
	base := upstream.Scheme + "://" + upstream.Host
	if strings.HasPrefix(loc, base) {
		suffix := strings.TrimPrefix(loc, base)
		if suffix == "" || suffix == "/" {
			resp.Header.Set("Location", "/ipfs/")
			return
		}
		if !strings.HasPrefix(suffix, "/") {
			suffix = "/" + suffix
		}
		resp.Header.Set("Location", suffix)
		return
	}
	if !u.IsAbs() && (strings.HasPrefix(u.Path, "/ipfs/") || strings.HasPrefix(u.Path, "/ipns/")) {
		newLoc := u.Path
		if u.RawQuery != "" {
			newLoc += "?" + u.RawQuery
		}
		resp.Header.Set("Location", newLoc)
	}
}
