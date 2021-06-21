package proxy

import (
	"crypto/tls"
	"github.com/kataras/iris/v12/context"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var (
	keyPrefix           = "Bearer"
	AuthorizationHeader = "Authorization"
)

func KubernetesApiProxy(ctx *context.Context) {
	c := config.GetConfig()
	proxyPath := ctx.Params().Get("p")
	var urlStr string
	if c.KubeConfig != "" {
		cs, err := clientcmd.BuildConfigFromFlags("", c.KubeConfig)
		if err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			_, _ = ctx.JSON(err.Error())
			return
		}
		urlStr = cs.Host

	} else {
		urlStr = c.KubeApiServerConfig.ApiServer
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		_, _ = ctx.JSON(err.Error())
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	if c.KubeConfig != "" {
		cs, err := clientcmd.BuildConfigFromFlags("", c.KubeConfig)
		if err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			_, _ = ctx.JSON(err.Error())
			return
		}
		ts, err := rest.TransportFor(cs)
		if err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			_, _ = ctx.JSON(err.Error())
			return
		}
		proxy.Transport = ts
	} else {
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		token := fmt.Sprintf("%s %s", keyPrefix, c.Token)
		ctx.Request().Header.Add(AuthorizationHeader, token)
	}
	ctx.Request().URL.Path = proxyPath
	logrus.Infof("[%s] %s", ctx.Request().Method, ctx.Request().URL.String())
	proxy.ServeHTTP(ctx.ResponseWriter(), ctx.Request())
}
