// Package std embute a biblioteca padrao escrita em LIAF.
//
// Os modulos vivem aqui, como arquivos .liaf, e viajam dentro do liafc:
// (import "std/auth") nunca le o disco nem a rede. A versao de cada modulo e
// sempre a do compilador que o carregou, entao nao existe versao para um
// modelo errar nem ambiente para quebrar.
package std

import "embed"

//go:embed *.liaf
var FS embed.FS
