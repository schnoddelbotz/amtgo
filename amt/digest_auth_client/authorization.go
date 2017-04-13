package digestAuthClient

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"net/url"
	"strings"
	"time"
)

type authorization struct {
	Algorithm string // unquoted
	Cnonce    string // quoted
	Nc        int    // unquoted
	Nonce     string // quoted
	Opaque    string // quoted
	Qop       string // unquoted
	Realm     string // quoted
	Response  string // quoted
	URI       string // quoted
	Userhash  bool   // quoted
	Username  string // quoted
	//Username_ string // quoted
}

func newAuthorization(dr *DigestRequest) (*authorization, error) {

	ah := authorization{
		Algorithm: dr.Wa.Algorithm,
		Cnonce:    "",
		Nc:        0,
		Nonce:     dr.Wa.Nonce,
		Opaque:    dr.Wa.Opaque,
		Qop:       "",
		Realm:     dr.Wa.Realm,
		Response:  "",
		URI:       "",
		Userhash:  dr.Wa.Userhash,
		Username:  "",
		//Username_: "", // TODO
	}

	return ah.refreshAuthorization(dr)
}

func (ah *authorization) refreshAuthorization(dr *DigestRequest) (*authorization, error) {

	ah.Username = dr.Username

	if ah.Userhash {
		ah.Username = ah.hash(fmt.Sprintf("%s:%s", ah.Username, ah.Realm))
	}

	ah.Nc++

	ah.Cnonce = ah.hash(fmt.Sprintf("%d:%s:my_value", time.Now().UnixNano(), dr.Username))

	url, err := url.Parse(dr.URI)
	if err != nil {
		return nil, err
	}
	ah.URI = url.RequestURI()

	ah.Response = ah.computeResponse(dr)
	return ah, nil
}

func (ah *authorization) computeResponse(dr *DigestRequest) (s string) {
	kdSecret := ah.hash(ah.computeA1(dr))
	kdData := fmt.Sprintf("%s:%08x:%s:%s:%s", ah.Nonce, ah.Nc, ah.Cnonce, ah.Qop, ah.hash(ah.computeA2(dr)))
	return ah.hash(fmt.Sprintf("%s:%s", kdSecret, kdData))
}

func (ah *authorization) computeA1(dr *DigestRequest) string {
	return fmt.Sprintf("%s:%s:%s", ah.Username, ah.Realm, dr.Password)
}

func (ah *authorization) computeA2(dr *DigestRequest) string {
	ah.Qop = "auth"
	return fmt.Sprintf("%s:%s", dr.Method, ah.URI)
}

func (ah *authorization) hash(a string) string {
	var h hash.Hash
	h = md5.New()
	io.WriteString(h, a)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (ah *authorization) toString() string {
	var buffer bytes.Buffer

	buffer.WriteString("Digest ")

	if ah.Username != "" {
		buffer.WriteString(fmt.Sprintf("username=\"%s\", ", ah.Username))
	}

	if ah.Realm != "" {
		buffer.WriteString(fmt.Sprintf("realm=\"%s\", ", ah.Realm))
	}

	if ah.Algorithm != "" {
		buffer.WriteString(fmt.Sprintf("algorithm=%s, ", ah.Algorithm))
	}

	if ah.Nonce != "" {
		buffer.WriteString(fmt.Sprintf("nonce=\"%s\", ", ah.Nonce))
	}

	if ah.URI != "" {
		buffer.WriteString(fmt.Sprintf("uri=\"%s\", ", ah.URI))
	}

	if ah.Cnonce != "" {
		buffer.WriteString(fmt.Sprintf("cnonce=\"%s\", ", ah.Cnonce))
	}

	if ah.Nc != 0 {
		buffer.WriteString(fmt.Sprintf("nc=%08x, ", ah.Nc))
	}

	if ah.Qop != "" {
		buffer.WriteString(fmt.Sprintf("qop=%s, ", ah.Qop))
	}

	if ah.Opaque != "" {
		buffer.WriteString(fmt.Sprintf("opaque=\"%s\", ", ah.Opaque))
	}

	if ah.Response != "" {
		buffer.WriteString(fmt.Sprintf("response=\"%s\", ", ah.Response))
	}

	if ah.Userhash {
		buffer.WriteString("userhash=true, ")
	}

	s := buffer.String()
	return strings.TrimSuffix(s, ", ")
}
