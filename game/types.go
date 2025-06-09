package game

type PlayerState struct {
	ID   string
	X, Y int
}

// GameState representa o estado completo do jogo com todos os jogadores.
// É esta estrutura que o servidor envia para os clientes.
type GameState struct {
	Players map[string]PlayerState
}

// ClientCommand é o comando que um cliente envia para o servidor.
type ClientCommand struct {
	ClientID       string                 // Para o servidor saber quem enviou o comando
	SequenceNumber int64                  // Para garantir a idempotência (execução "exactly-once")
	Action         string                 // "move", "interact", etc.
	Params         map[string]interface{} // Dados extras, como a direção do movimento
}

// EmptyArgs é usada para chamadas RPC que não precisam de argumentos.
// Isto torna a assinatura do método explícita e evita erros de "type mismatch".
type EmptyArgs struct{}

