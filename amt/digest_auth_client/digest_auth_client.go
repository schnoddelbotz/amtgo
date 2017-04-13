// Package digestAuthClient implements HTTP digest auth for AMT.
// It is based on https://github.com/xinsnake/go-http-digest-auth-client,
// but was adapted for AMT usage: makes timeouts configurable, makes
// TLS cert verification configurable, closes connections...
package digestAuthClient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"time"
)

// DigestRequest describes a HTTP digest auth request
type DigestRequest struct {
	Body       string
	Method     string
	Password   string
	URI        string
	Username   string
	Timeout    time.Duration
	SkipCert   bool
	CaCertData []byte
	Auth       *authorization
	Wa         *wwwAuthenticate
}

// NewRequest returns a new DigestRequest
func NewRequest(username string, password string, method string, uri string, body string, timeout time.Duration, skipVerify bool, caCertData []byte) DigestRequest {
	dr := DigestRequest{}
	dr.UpdateRequest(username, password, method, uri, body, timeout, skipVerify, caCertData)
	return dr
}

// UpdateRequest updates an existing DigestRequest
func (dr *DigestRequest) UpdateRequest(username string,
	password string, method string, uri string, body string, timeout time.Duration, skipVerify bool, caCertData []byte) *DigestRequest {

	dr.Body = body
	dr.Method = method
	dr.Password = password
	dr.URI = uri
	dr.Username = username
	dr.Timeout = timeout
	dr.SkipCert = skipVerify
	dr.CaCertData = caCertData
	return dr
}

// Execute executes as DigestRequest
func (dr *DigestRequest) Execute() (resp *http.Response, err error) {
	if dr.Auth == nil {
		var req *http.Request
		if req, err = http.NewRequest(dr.Method, dr.URI, bytes.NewReader([]byte(dr.Body))); err != nil {
			return nil, err
		}
		req.Close = true

		tr := &http.Transport{
			DialTLS: func(network, addr string) (net.Conn, error) {
				return tls.DialWithDialer(&net.Dialer{Timeout: dr.Timeout}, network, addr,
					&tls.Config{MaxVersion: tls.VersionTLS10, InsecureSkipVerify: true})
			},
			IdleConnTimeout: 5 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   dr.Timeout,
				KeepAlive: 5 * time.Second,
				DualStack: false,
			}).DialContext,
		}

		client := &http.Client{
			Timeout: dr.Timeout,
		}
		if !dr.SkipCert && len(dr.CaCertData) > 0 {
			// enable TLS CA cert verification
			roots := x509.NewCertPool()
			ok := roots.AppendCertsFromPEM(dr.CaCertData)
			if !ok {
				panic("failed to parse root certificate")
			}
			tr.TLSClientConfig = &tls.Config{MaxVersion: tls.VersionTLS10, RootCAs: roots}
			tr.DialTLS = func(network, addr string) (net.Conn, error) {
				return tls.DialWithDialer(&net.Dialer{Timeout: dr.Timeout}, network, addr,
					&tls.Config{MaxVersion: tls.VersionTLS10, InsecureSkipVerify: false, RootCAs: roots})
			}
		}
		client.Transport = tr
		req.Header.Set("Connection", "close")

		//debug(httputil.DumpRequestOut(req, true))
		resp, err = client.Do(req)
		if err != nil {
			return nil, err
		}
		//debug(httputil.DumpResponse(resp, true))

		if resp.StatusCode == 401 {
			resp.Body.Close()
			return dr.executeNewDigest(resp)
		}
		return
	}

	return dr.executeExistingDigest()
}

func (dr *DigestRequest) executeNewDigest(resp *http.Response) (*http.Response, error) {
	var (
		auth *authorization
		err  error
		wa   *wwwAuthenticate
	)

	waString := resp.Header.Get("WWW-Authenticate")
	if waString == "" {
		return nil, fmt.Errorf("Failed to get WWW-Authenticate header, please check your server configuration")
	}
	wa = newWwwAuthenticate(waString)
	dr.Wa = wa

	if auth, err = newAuthorization(dr); err != nil {
		//fmt.Printf("ERR WITH newAuth: %s\n", err)
		return nil, err
	}
	authString := auth.toString()

	var r *http.Response
	if r, err = dr.executeRequest(authString); err != nil {
		//fmt.Printf("ERR WITH drExec: %s\n", err)
		return nil, err
	}
	dr.Auth = auth
	//debug(httputil.DumpResponse(resp, true))
	return r, nil
}

func (dr *DigestRequest) executeExistingDigest() (*http.Response, error) {
	var (
		auth *authorization
		err  error
	)

	if auth, err = dr.Auth.refreshAuthorization(dr); err != nil {
		return nil, err
	}
	dr.Auth = auth

	authString := dr.Auth.toString()
	return dr.executeRequest(authString)
}

func (dr *DigestRequest) executeRequest(authString string) (*http.Response, error) {
	var (
		err error
		req *http.Request
	)

	if req, err = http.NewRequest(dr.Method, dr.URI, bytes.NewReader([]byte(dr.Body))); err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", authString)
	req.Header.Add("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("Connection", "close")
	req.Close = true

	tr := &http.Transport{
		DialTLS: func(network, addr string) (net.Conn, error) {
			return tls.DialWithDialer(&net.Dialer{Timeout: dr.Timeout}, network, addr,
				&tls.Config{MaxVersion: tls.VersionTLS10, InsecureSkipVerify: true})
		},
		IdleConnTimeout: 5 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   dr.Timeout,
			KeepAlive: 5 * time.Second,
			DualStack: false,
		}).DialContext,
	}

	client := &http.Client{
		Timeout: dr.Timeout,
	}
	if !dr.SkipCert && len(dr.CaCertData) > 0 {
		// enable TLS CA cert verification
		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM(dr.CaCertData)
		if !ok {
			panic("failed to parse root certificate")
		}
		tr.DialTLS = func(network, addr string) (net.Conn, error) {
			return tls.DialWithDialer(&net.Dialer{Timeout: dr.Timeout}, network, addr,
				&tls.Config{MaxVersion: tls.VersionTLS10, InsecureSkipVerify: false, RootCAs: roots})
		}
	}

	client.Transport = tr
	//debug(httputil.DumpRequestOut(req, true))
	return client.Do(req)
}

func debug(data []byte, err error) {
	if err == nil {
		fmt.Printf("%s\n------------------------------------------------------\n", data)
	} else {
		fmt.Printf("%s\n============ errrr ^^^ ===============================\n", err)
	}
}
