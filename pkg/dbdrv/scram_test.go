package dbdrv

import "testing"

// Vetor de teste da RFC 7677, secao 3. Exercita PBKDF2-HMAC-SHA256, o calculo
// do ClientProof e a verificacao da assinatura do servidor de uma vez so; se
// qualquer um dos tres estiver errado, a prova nao bate com a do RFC.
//
// O nonce e o firstBare sao fixados porque newSCRAMClient sorteia o nonce e
// usa n= vazio (o PostgreSQL manda o usuario no StartupMessage), enquanto o
// vetor do RFC usa n=user.
const (
	rfcServerFirst = "r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096"
	rfcClientFinal = "c=biws,r=rOprNGfwEbeRWgbNEkqO%hvYDpWUa2RaTCAfuxFIlj)hNlF$k0,p=dHzbZapWIk4jUhN+Ute9ytag9zjfMHgsqmmiz7AndVQ="
	rfcServerFinal = "v=6rriTRBi23WpRR/wtup+mMhUZUn/dB5nLTJRsjl95G4="
)

func rfcClient() *scramClient {
	return &scramClient{
		password:  "pencil",
		nonce:     "rOprNGfwEbeRWgbNEkqO",
		firstBare: "n=user,r=rOprNGfwEbeRWgbNEkqO",
	}
}

func TestSCRAMMatchesRFC7677(t *testing.T) {
	c := rfcClient()
	final, err := c.final(rfcServerFirst)
	if err != nil {
		t.Fatalf("final: %v", err)
	}
	if final != rfcClientFinal {
		t.Fatalf("client-final divergiu do RFC 7677\n got: %s\nwant: %s", final, rfcClientFinal)
	}
	if err := c.verify(rfcServerFinal); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestSCRAMRejectsForgedServerSignature(t *testing.T) {
	c := rfcClient()
	if _, err := c.final(rfcServerFirst); err != nil {
		t.Fatalf("final: %v", err)
	}
	// Sem esta recusa a autenticacao deixaria de ser mutua e um intermediario
	// poderia se passar pelo banco.
	if err := c.verify("v=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="); err == nil {
		t.Fatal("assinatura forjada do servidor foi aceita")
	}
}

func TestSCRAMRejectsServerNonce(t *testing.T) {
	for _, tc := range []struct{ name, serverFirst string }{
		{"nonce nao estende o do cliente", "r=outroNonceQualquer,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096"},
		{"sem nonce", "s=W22ZaJ0SNY7soEsUEjb6gQ==,i=4096"},
		{"iteracoes invalidas", "r=rOprNGfwEbeRWgbNEkqOxx,s=W22ZaJ0SNY7soEsUEjb6gQ==,i=0"},
		{"salt invalido", "r=rOprNGfwEbeRWgbNEkqOxx,s=nao-e-base64!!,i=4096"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := rfcClient().final(tc.serverFirst); err == nil {
				t.Fatal("esperado erro, recebido sucesso")
			}
		})
	}
}

func TestSCRAMReportsServerError(t *testing.T) {
	c := rfcClient()
	if _, err := c.final(rfcServerFirst); err != nil {
		t.Fatalf("final: %v", err)
	}
	if err := c.verify("e=invalid-proof"); err == nil {
		t.Fatal("erro do servidor foi tratado como sucesso")
	}
}

func TestSCRAMNonceIsFresh(t *testing.T) {
	a, err := newSCRAMClient("x")
	if err != nil {
		t.Fatal(err)
	}
	b, err := newSCRAMClient("x")
	if err != nil {
		t.Fatal(err)
	}
	if a.nonce == b.nonce {
		t.Fatal("dois clientes sortearam o mesmo nonce")
	}
	if got := a.first(); got[:3] != "n,," {
		t.Fatalf("client-first sem o gs2-header: %q", got)
	}
}
