package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

type Server struct {
	in     *bufio.Reader
	out    io.Writer
	outMu  sync.Mutex
	docsMu sync.RWMutex
	docs   map[string]string // URI -> conteúdo do arquivo
	isShut bool
}

func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{
		in:   bufio.NewReader(in),
		out:  out,
		docs: make(map[string]string),
	}
}

// Serve executa o loop de leitura e despacho de mensagens JSON-RPC até EOF ou exit.
func (s *Server) Serve() error {
	for {
		content, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		var req RequestMessage
		if err := json.Unmarshal(content, &req); err != nil {
			continue
		}

		if req.ID != nil {
			// É uma requisição que espera resposta
			go s.handleRequest(req)
		} else {
			// É uma notificação
			go s.handleNotification(req.Method, req.Params)
		}
	}
}

func (s *Server) readMessage() ([]byte, error) {
	contentLength := 0

	for {
		line, err := s.in.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			// Linha em branco separa o cabeçalho do payload
			break
		}

		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if n, err := strconv.Atoi(val); err == nil {
					contentLength = n
				}
			}
		}
	}

	if contentLength <= 0 {
		return nil, fmt.Errorf("cabeçalho Content-Length ausente ou inválido")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(s.in, body); err != nil {
		return nil, err
	}

	return body, nil
}

func (s *Server) writeMessage(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))

	s.outMu.Lock()
	defer s.outMu.Unlock()

	if _, err := s.out.Write([]byte(header)); err != nil {
		return err
	}
	if _, err := s.out.Write(data); err != nil {
		return err
	}
	return nil
}

func (s *Server) sendResponse(id any, result any, respErr *ResponseError) {
	resp := ResponseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   respErr,
	}
	_ = s.writeMessage(resp)
}

func (s *Server) sendNotification(method string, params any) {
	notif := NotificationMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	_ = s.writeMessage(notif)
}

func (s *Server) handleRequest(req RequestMessage) {
	switch req.Method {
	case "initialize":
		res := InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync: 1, // Full sync
				CompletionProvider: &CompletionOptions{
					TriggerCharacters: []string{".", "-", "("},
				},
				HoverProvider:              true,
				DocumentFormattingProvider: true,
			},
			ServerInfo: ServerInfo{
				Name:    "liaf-lsp",
				Version: "1.0.0",
			},
		}
		s.sendResponse(req.ID, res, nil)

	case "shutdown":
		s.isShut = true
		s.sendResponse(req.ID, nil, nil)

	case "textDocument/completion":
		var params CompletionParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendResponse(req.ID, nil, &ResponseError{Code: -32602, Message: err.Error()})
			return
		}
		s.docsMu.RLock()
		content := s.docs[params.TextDocument.URI]
		s.docsMu.RUnlock()

		list := computeCompletions(content, params.Position)
		s.sendResponse(req.ID, list, nil)

	case "textDocument/hover":
		var params HoverParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendResponse(req.ID, nil, &ResponseError{Code: -32602, Message: err.Error()})
			return
		}
		s.docsMu.RLock()
		content := s.docs[params.TextDocument.URI]
		s.docsMu.RUnlock()

		hover := computeHover(content, params.Position)
		s.sendResponse(req.ID, hover, nil)

	case "textDocument/formatting":
		var params DocumentFormattingParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendResponse(req.ID, nil, &ResponseError{Code: -32602, Message: err.Error()})
			return
		}
		s.docsMu.RLock()
		content := s.docs[params.TextDocument.URI]
		s.docsMu.RUnlock()

		edits := computeFormatting(content)
		s.sendResponse(req.ID, edits, nil)

	default:
		s.sendResponse(req.ID, nil, &ResponseError{
			Code:    -32601,
			Message: fmt.Sprintf("Método não implementado: %s", req.Method),
		})
	}
}

func (s *Server) handleNotification(method string, rawParams json.RawMessage) {
	switch method {
	case "initialized":
		// Confirmação de handshake recebida, pronto para operar.

	case "textDocument/didOpen":
		var params DidOpenTextDocumentParams
		if err := json.Unmarshal(rawParams, &params); err == nil {
			s.docsMu.Lock()
			s.docs[params.TextDocument.URI] = params.TextDocument.Text
			s.docsMu.Unlock()
			s.publishDiagnostics(params.TextDocument.URI, params.TextDocument.Text)
		}

	case "textDocument/didChange":
		var params DidChangeTextDocumentParams
		if err := json.Unmarshal(rawParams, &params); err == nil && len(params.ContentChanges) > 0 {
			// Com full sync (TextDocumentSync: 1), o último item contém o texto completo do arquivo
			newText := params.ContentChanges[len(params.ContentChanges)-1].Text
			s.docsMu.Lock()
			s.docs[params.TextDocument.URI] = newText
			s.docsMu.Unlock()
			s.publishDiagnostics(params.TextDocument.URI, newText)
		}

	case "textDocument/didClose":
		var params DidCloseTextDocumentParams
		if err := json.Unmarshal(rawParams, &params); err == nil {
			s.docsMu.Lock()
			delete(s.docs, params.TextDocument.URI)
			s.docsMu.Unlock()
			// Limpa os diagnósticos ao fechar o documento
			s.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
				URI:         params.TextDocument.URI,
				Diagnostics: []Diagnostic{},
			})
		}
	}
}

func (s *Server) publishDiagnostics(uri, content string) {
	diags := computeDiagnostics(content, uri)
	s.sendNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	})
}
