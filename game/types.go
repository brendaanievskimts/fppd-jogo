package game

// Elemento representa um objeto do mapa
type Elemento struct {
    Simbolo  rune
    Tangivel bool // Bloqueia passagem?
}

// PlayerState guarda a informação de um único jogador
type PlayerState struct {
    ID           string
    X, Y         int
    GramasComidas int
}

// GameState é a fotografia completa do estado do jogo em um dado momento
type GameState struct {
    Mapa    [][]Elemento
    Players map[string]PlayerState
    Status  string // Mensagem do jogo (ex: tempo, vencedor)
}

// ClientCommand é o comando enviado do cliente para o servidor
type ClientCommand struct {
    ClientID       string
    SequenceNumber int64
    Action         string
    Params         map[string]interface{}
}

// EmptyArgs é usado para chamadas RPC que não precisam de argumentos
type EmptyArgs struct{}