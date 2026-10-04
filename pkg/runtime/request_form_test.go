package runtime

import (
	"net/url"
	"testing"

	"liaf/pkg/web"
)

func TestRequestFormDecodesBodyAndFallsBackToQuery(t *testing.T) {
	req := web.Request{
		Body:  "titulo=Caf%C3%A9+com+leite&dono=ana",
		Query: url.Values{"filtro": {"urgente"}},
	}

	if r := RequestForm(req, "titulo"); !r.OK || r.Value != "Café com leite" {
		t.Errorf("titulo: %+v", r)
	}
	if r := RequestForm(req, "filtro"); !r.OK || r.Value != "urgente" {
		t.Errorf("fallback para query: %+v", r)
	}
	if r := RequestForm(req, "ausente"); r.OK {
		t.Errorf("campo ausente devia falhar: %+v", r)
	}
}
