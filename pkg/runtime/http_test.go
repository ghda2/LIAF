package runtime

import (
	"net/http"
	"net/url"
	"testing"

	"liaf/pkg/web"
)

func TestRequestHeader(t *testing.T) {
	req := web.Request{Header: http.Header{}}
	req.Header.Set("Authorization", "Bearer abc")
	req.Header.Add("X-Tenant", "")
	req.Header.Add("Accept", "text/html")
	req.Header.Add("Accept", "application/json")

	for _, tc := range []struct {
		name string
		ok   bool
		want string
	}{
		{"authorization", true, "Bearer abc"}, // sem diferenciar maiusculas
		{"X-Tenant", true, ""},                // presente e vazio nao e ausente
		{"Accept", true, "text/html"},         // repetido: o primeiro
		{"Cookie", false, ""},
	} {
		r := RequestHeader(req, tc.name)
		if r.OK != tc.ok || r.Value != tc.want {
			t.Errorf("%s: %+v", tc.name, r)
		}
	}
	if r := RequestHeader(web.Request{}, "Authorization"); r.OK {
		t.Error("request sem headers devolveu ok")
	}
}

func TestRequestQuery(t *testing.T) {
	q, _ := url.ParseQuery("status=aberto&status=pronto&pagina=&Tipo=x")
	req := web.Request{Query: q}

	for _, tc := range []struct {
		name string
		ok   bool
		want string
	}{
		{"status", true, "aberto"},
		{"pagina", true, ""},
		{"tipo", false, ""}, // a URL diferencia maiusculas
		{"ausente", false, ""},
	} {
		r := RequestQuery(req, tc.name)
		if r.OK != tc.ok || r.Value != tc.want {
			t.Errorf("%s: %+v", tc.name, r)
		}
	}
	if r := RequestQuery(web.Request{}, "status"); r.OK {
		t.Error("request sem query devolveu ok")
	}
}

func TestRequestCookie(t *testing.T) {
	req := web.Request{Header: http.Header{}}
	req.Header.Add("Cookie", "session=xyz123; user_id=42; theme=dark")

	if r := RequestCookie(req, "session"); !r.OK || r.Value != "xyz123" {
		t.Errorf("cookie session: %+v", r)
	}
	if r := RequestCookie(req, "user_id"); !r.OK || r.Value != "42" {
		t.Errorf("cookie user_id: %+v", r)
	}
	if r := RequestCookie(req, "theme"); !r.OK || r.Value != "dark" {
		t.Errorf("cookie theme: %+v", r)
	}
	if r := RequestCookie(req, "non_existent"); r.OK {
		t.Errorf("cookie inexistente devolveu ok")
	}
}

func TestResponseAddHeader(t *testing.T) {
	res := web.Response{Status: 200, Body: "ok"}
	res = ResponseAddHeader(res, "Set-Cookie", "a=1")
	res = ResponseAddHeader(res, "Set-Cookie", "b=2")
	res = ResponseAddHeader(res, "Vary", "Origin")
	res = ResponseAddHeader(res, "Vary", "Accept-Encoding")

	if len(res.Headers) != 4 {
		t.Fatalf("esperado 4 headers, obteve %d", len(res.Headers))
	}
}
