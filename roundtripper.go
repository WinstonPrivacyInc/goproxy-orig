package goproxy

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"
	//"net/http/httptrace"
	"github.com/winston/shadownetwork"
	"context"
	//"errors"
)

type RoundTripper interface {
	RoundTrip(req *http.Request, ctx *ProxyCtx) (*http.Response, error)
}

type RoundTripperFunc func(req *http.Request, ctx *ProxyCtx) (*http.Response, error)

func (f RoundTripperFunc) RoundTrip(req *http.Request, ctx *ProxyCtx) (*http.Response, error) {
	return f(req, ctx)
}

func (ctx *ProxyCtx) RoundTrip(req *http.Request) (*http.Response, error) {
	// RLS 2/16/2018 - This is where requests are made to the original destination sites.

	var tr http.RoundTripper
	var addendum = []string{""}

	requestcontext := req.Context()

	// Redirect with Fake Destination ?
	if ctx.RoundTripper == nil {
		if ctx.fakeDestinationDNS != "" {
			req.URL.Host = ctx.fakeDestinationDNS
			transport := &http.Transport{
				TLSClientConfig: &tls.Config{
					ServerName:         strings.Split(ctx.host, ":")[0],
					InsecureSkipVerify: true,
				},
				Proxy: ctx.Proxy.Transport.Proxy,
			}
			addendum = append(addendum, fmt.Sprintf(", sni=%q, fakedns=%q", transport.TLSClientConfig.ServerName, ctx.fakeDestinationDNS))
			tr = transport
		} else {
			if ctx.PrivateNetwork && ctx.Proxy.PrivateNetwork != nil {
				ctx.ShadowTransport = ctx.Proxy.PrivateNetwork.Transport()
				if ctx.ShadowTransport == nil {
					// ShadowTransport was nil for some reason. Use local transport.
					//if strings.Contains(req.URL.String(), "whatis") {
						//fmt.Printf("  *** RoundTrip() ShadowTransport was nil - %s\n", req.URL)
					//}
					tr = ctx.Proxy.Transport

					// Ensures we report the correct cloaked status back to the caller
					ctx.PrivateNetwork = false
				} else {
					// Point to the shadow transport so we can pass values to and from http.Request

					//if strings.Contains(req.URL.String(), "whatis") {

					//}
					requestcontext = context.WithValue(requestcontext, shadownetwork.ShadowTransportKey, ctx.ShadowTransport)

					tr = ctx.ShadowTransport.Transport
					//fmt.Printf("  *** RoundTrip() successfully hooked into Shadow Transport - %s \n %+v\n", req.URL, ctx.ShadowTransport)
				}

			} else {
				//fmt.Printf("  *** Using local transport")
				tr = ctx.Proxy.Transport

				// Ensures we report the correct cloaked status back to the caller
				ctx.PrivateNetwork = false
			}
		}

		if ctx.Proxy.FlushIdleConnections {
			ctx.Proxy.Transport.CloseIdleConnections()
			ctx.Proxy.FlushIdleConnections = false
		}

		ctx.RoundTripper = ctx.wrapTransport(tr)
	}

	if ctx.isLogEnabled {
		addendum = append(addendum, "log=yes")
	}

	resp, err := ctx._roundTripWithLog(req.WithContext(requestcontext))
	//resp, err := ctx._roundTripWithLog(req)


	return resp, err
}

func (ctx *ProxyCtx) _roundTripWithLog(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	if ctx.isLogEnabled == true {
		reqAndResp := new(harReqAndResp)
		reqAndResp.start = time.Now()
		reqAndResp.captureContent = ctx.isLogWithContent

		req := ctx.Req
		if reqAndResp.captureContent && req.ContentLength > 0 {
			req, reqAndResp.req = copyReq(req)
		} else {
			reqAndResp.req = req
		}

		resp, err = ctx.RoundTripper.RoundTrip(req, ctx)

		if reqAndResp.captureContent && resp != nil && resp.ContentLength != 0 {
			resp, reqAndResp.resp = copyResp(resp)
		} else {
			reqAndResp.resp = resp
		}

		reqAndResp.end = time.Now()
		ctx.Proxy.harLogEntryCh <- *reqAndResp

	} else {
		resp, err = ctx.RoundTripper.RoundTrip(req, ctx)
	}

	return resp, err
}

func (ctx *ProxyCtx) wrapTransport(tr http.RoundTripper) RoundTripper {
	return RoundTripperFunc(func(req *http.Request, ctx *ProxyCtx) (*http.Response, error) {

		// Add tracing to a specific domain. This is really helpful in debugging connection issues.
		/*if strings.Contains(req.URL.String(), "xaxis") {
			trace := &httptrace.ClientTrace{
				// A private network request does not currently trigger DNS lookups because these
				// are resolved by the server.
				DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
					fmt.Printf("DNS Info: %+v\n", dnsInfo)
				},
				GotConn: func(connInfo httptrace.GotConnInfo) {
					fmt.Printf("Got Conn: %+v\n", connInfo)
				},
			}
			req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
			//fmt.Printf("  *** wrapTransport: %s\n", req.URL.String())
		}*/

		//fmt.Println("[DEBUG] GoProxy.RoundTripper() Start")
		resp, err := tr.RoundTrip(req)

		if ctx.Trace {
			fmt.Printf("  *** RoundTrip(): err: %+v\n  resp: %s\n", err, resp)
		}

		// Check for shadow network errors

		if ctx.PrivateNetwork && ctx.ShadowTransport != nil {
			if err != nil {
				// We received an error but the private transport failed over to the local (uncloaked) transport.
				// Use this response to minimize any disruption.
				if resp != nil {
					fmt.Printf("[DEBUG] RoundTrip() - received response and an error. Using response anyway. %+v\n", err)
					ctx.PrivateNetwork = false
					err = nil
				}
			}

			// Check to see if we connected but the remote node actively refused the connection
			/*if err != nil && err.Error() == "n=0 socket close" {
				//ctx.ShadowTransport.RecordFailedConnection(shadownetwork.ErrorRefusedConnection)
				err = errors.New(shadownetwork.ErrorConnectionFailed)
				ctx.PrivateNetwork = false

			} else {
				// Check for network connectivity errors (the remote node doesn't exist)
				dnsbypassctx := req.Context()

				errmsg := dnsbypassctx.Value(shadownetwork.ShadowTransportFailed)

				//fmt.Println("Error message", errmsg)
				if (errmsg != nil) {
					errmsgstruct := errmsg.(*shadownetwork.ShadowNetworkFailure)
					if errmsgstruct.Failed {
						ctx.PrivateNetwork = false
					}
				}
			}*/
		}

		//fmt.Println("[DEBUG] GoProxy.RoundTripper() End")
		return resp, err
	})
}
