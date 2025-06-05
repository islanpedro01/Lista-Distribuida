// internal/remotelist/types.go
package remotelist

// --- Append ---
// Args para inserir um valor ao final de uma lista.
type AppendArgs struct {
	ListID string // identificador da lista
	Value  int    // valor a ser inserido
}

// (poderíamos devolver bool, mas normalmente retornamos error se der ruim)
type AppendReply struct {
	Success bool // true se funcionou
}

// --- Get ---
// Args para obter o i-ésimo elemento de uma lista.
type GetArgs struct {
	ListID string // identificador da lista
	Index  int    // posição desejada
}

type GetReply struct {
	Value int   // valor retornado
	Err   error // se der erro (fora de intervalo ou lista não existe)
}

// --- Remove ---
// Args para remover o último elemento
type RemoveArgs struct {
	ListID string // identificador da lista
}

type RemoveReply struct {
	Value int   // valor que foi removido
	Err   error // se der erro (lista vazia ou não existe)
}

// --- Size ---
// Args para obter o tamanho da lista
type SizeArgs struct {
	ListID string // identificador da lista
}

type SizeReply struct {
	Size int // número de elementos na lista
}
