package lsp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type mockClient struct {
	inPipe  *io.PipeWriter
	outPipe *io.PipeReader
	mu      sync.Mutex
}

func newMockClient() (*mockClient, io.Reader, io.Writer) {
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()

	client := &mockClient{
		inPipe:  clientOut,
		outPipe: clientIn,
	}
	return client, serverIn, serverOut
}

func (c *mockClient) send(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, err := c.inPipe.Write([]byte(header)); err != nil {
		return err
	}
	_, err = c.inPipe.Write(data)
	return err
}

func (c *mockClient) readResponse(timeout time.Duration) ([]byte, error) {
	type res struct {
		data []byte
		err  error
	}
	ch := make(chan res, 1)

	go func() {
		contentLength := 0
		var headerBuf bytes.Buffer
		for {
			b := make([]byte, 1)
			if _, err := c.outPipe.Read(b); err != nil {
				ch <- res{nil, err}
				return
			}
			headerBuf.Write(b)
			if strings.HasSuffix(headerBuf.String(), "\r\n\r\n") {
				headerStr := headerBuf.String()
				for _, line := range strings.Split(headerStr, "\r\n") {
					if strings.HasPrefix(strings.ToLower(line), "content-length:") {
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							n, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
							contentLength = n
						}
					}
				}
				break
			}
		}

		if contentLength <= 0 {
			ch <- res{nil, fmt.Errorf("tamanho invalido")}
			return
		}

		body := make([]byte, contentLength)
		if _, err := io.ReadFull(c.outPipe, body); err != nil {
			ch <- res{nil, err}
			return
		}
		ch <- res{body, nil}
	}()

	select {
	case r := <-ch:
		return r.data, r.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout aguardando resposta")
	}
}

func TestLSPLifecycle(t *testing.T) {
	client, serverIn, serverOut := newMockClient()
	server := NewServer(serverIn, serverOut)

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Serve()
	}()

	// 1. Initialize
	initReq := RequestMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{"processId": 1234}`),
	}
	if err := client.send(initReq); err != nil {
		t.Fatalf("erro ao enviar initialize: %v", err)
	}

	resBytes, err := client.readResponse(3 * time.Second)
	if err != nil {
		t.Fatalf("erro ao ler resposta de initialize: %v", err)
	}

	var initResp ResponseMessage
	if err := json.Unmarshal(resBytes, &initResp); err != nil {
		t.Fatalf("erro ao deserializar initialize: %v", err)
	}
	if initResp.Error != nil {
		t.Fatalf("initialize retornou erro: %v", initResp.Error)
	}

	// 2. DidOpen com erro para receber diagnósticos
	liafSourceWithError := `
fn main() void
  let x int = "erro de tipo"
end
`
	didOpen := NotificationMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didOpen",
		Params: DidOpenTextDocumentParams{
			TextDocument: TextDocumentItem{
				URI:        "file:///test.liaf",
				LanguageID: "liaf",
				Version:    1,
				Text:       liafSourceWithError,
			},
		},
	}
	if err := client.send(didOpen); err != nil {
		t.Fatalf("erro ao enviar didOpen: %v", err)
	}

	diagBytes, err := client.readResponse(3 * time.Second)
	if err != nil {
		t.Fatalf("erro ao ler publishDiagnostics: %v", err)
	}
	var diagNotif NotificationMessage
	if err := json.Unmarshal(diagBytes, &diagNotif); err != nil {
		t.Fatalf("erro ao deserializar publishDiagnostics: %v", err)
	}
	if diagNotif.Method != "textDocument/publishDiagnostics" {
		t.Fatalf("metodo inesperado: %s", diagNotif.Method)
	}

	// 3. Completion
	compReq := RequestMessage{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "textDocument/completion",
		Params:  json.RawMessage(`{"textDocument":{"uri":"file:///test.liaf"},"position":{"line":1,"character":2}}`),
	}
	if err := client.send(compReq); err != nil {
		t.Fatalf("erro ao enviar completion: %v", err)
	}
	compBytes, err := client.readResponse(3 * time.Second)
	if err != nil {
		t.Fatalf("erro ao ler completion: %v", err)
	}
	var compResp struct {
		Result CompletionList `json:"result"`
	}
	if err := json.Unmarshal(compBytes, &compResp); err != nil {
		t.Fatalf("erro ao deserializar completion: %v", err)
	}
	if len(compResp.Result.Items) == 0 {
		t.Fatalf("completion retornou lista vazia")
	}

	// Verifica se builtins essenciais estão na lista de completions
	foundPrintln := false
	for _, item := range compResp.Result.Items {
		if item.Label == "println" {
			foundPrintln = true
			break
		}
	}
	if !foundPrintln {
		t.Errorf("builtin 'println' nao encontrado no completion")
	}

	// 4. Hover sobre 'println'
	validSource := "fn main() void\nprintln(\"ola mundo\")\nend\n"
	_ = client.send(NotificationMessage{
		JSONRPC: "2.0",
		Method:  "textDocument/didChange",
		Params: DidChangeTextDocumentParams{
			TextDocument: VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: TextDocumentIdentifier{URI: "file:///test.liaf"},
				Version:                2,
			},
			ContentChanges: []TextDocumentContentChangeEvent{
				{Text: validSource},
			},
		},
	})
	// consome notificação de diagnósticos limpos do didChange
	_, _ = client.readResponse(3 * time.Second)

	hoverReq := RequestMessage{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "textDocument/hover",
		Params:  json.RawMessage(`{"textDocument":{"uri":"file:///test.liaf"},"position":{"line":1,"character":2}}`),
	}
	if err := client.send(hoverReq); err != nil {
		t.Fatalf("erro ao enviar hover: %v", err)
	}
	hoverBytes, err := client.readResponse(3 * time.Second)
	if err != nil {
		t.Fatalf("erro ao ler hover: %v", err)
	}
	var hoverResp struct {
		Result *Hover `json:"result"`
	}
	if err := json.Unmarshal(hoverBytes, &hoverResp); err != nil {
		t.Fatalf("erro ao deserializar hover: %v", err)
	}
	if hoverResp.Result == nil || !strings.Contains(hoverResp.Result.Contents.Value, "println") {
		t.Fatalf("hover esperado para println, obteve: %+v", hoverResp.Result)
	}

	// 5. Formatting
	formatReq := RequestMessage{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "textDocument/formatting",
		Params:  json.RawMessage(`{"textDocument":{"uri":"file:///test.liaf"},"options":{"tabSize":2,"insertSpaces":true}}`),
	}
	if err := client.send(formatReq); err != nil {
		t.Fatalf("erro ao enviar formatting: %v", err)
	}
	fmtBytes, err := client.readResponse(3 * time.Second)
	if err != nil {
		t.Fatalf("erro ao ler formatting: %v", err)
	}
	var fmtResp struct {
		Result []TextEdit `json:"result"`
	}
	if err := json.Unmarshal(fmtBytes, &fmtResp); err != nil {
		t.Fatalf("erro ao deserializar formatting: %v", err)
	}
	if len(fmtResp.Result) == 0 {
		t.Fatalf("formatting nao retornou edicoes de texto")
	}

	// 6. Shutdown
	shutReq := RequestMessage{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "shutdown",
	}
	_ = client.send(shutReq)
	_, _ = client.readResponse(3 * time.Second)

	// Fecha o pipe de entrada para encerrar o servidor
	_ = client.inPipe.Close()

	select {
	case err := <-serverErr:
		if err != nil {
			t.Fatalf("servidor terminou com erro: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("servidor nao finalizou apos fechar entrada")
	}
}
