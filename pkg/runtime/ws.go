package runtime

import (
	"liaf/pkg/web"
)

// Adaptadores dos builtins ws-* (issue #016). O enquadramento RFC 6455 e o
// pub/sub vivem em pkg/web, junto do servidor HTTP que faz o upgrade; aqui
// so se converte error em Result, que e a forma que a LIAF entende.

// WSSend envia uma mensagem de texto pela conexao.
func WSSend(c *web.WSConn, message string) Result[bool, string] {
	return Failure(c.Send(message))
}

// WSSendJSON serializa e envia numa chamada so.
func WSSendJSON(c *web.WSConn, value any) Result[bool, string] {
	return Failure(c.SendJSON(value))
}

// WSCloseConn envia o frame de fechamento com codigo e motivo e encerra.
func WSCloseConn(c *web.WSConn, code int64, reason string) Result[bool, string] {
	return Failure(c.CloseWith(int(code), reason))
}

// WSBroadcast entrega a mensagem aos inscritos no topico e devolve quantos a
// receberam. O tipo e (result int str) porque e o que a issue #016 define; a
// implementacao local em memoria nao tem caminho de falha, e um broker
// externo no futuro teria.
func WSBroadcast(topic, message string) Result[int64, string] {
	return Ok[int64, string](int64(web.WSBroadcast(topic, message)))
}

// WSJoin inscreve a conexao num topico para receber broadcasts.
func WSJoin(c *web.WSConn, topic string) { web.WSJoin(c, topic) }

// WSLeave cancela a inscricao sem fechar a conexao. Fechar ja desinscreve de
// todos os topicos, entao (on-close ...) nao precisa chamar isto.
func WSLeave(c *web.WSConn, topic string) { web.WSLeave(c, topic) }

// WSTopicSize informa quantos clientes estao inscritos no topico.
func WSTopicSize(topic string) int64 { return int64(web.WSTopicSize(topic)) }
