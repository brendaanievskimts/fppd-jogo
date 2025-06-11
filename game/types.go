package game

type ElementoDoMapa struct {
	Simbolo  rune
	Tangivel bool
}

//o nome já diz, estado do jogador
type PlayerState struct {
	Name                string 
	X, Y                int
	Vida                int
	VegetacoesColetadas int
}

//o estado que o jogo se encontra
type GameState struct {
	Mapa    [][]ElementoDoMapa
	Players map[string]PlayerState 
	Status  string
}

// JoinRequest é usado para o cliente se apresentar com o seu nome.
type JoinRequest struct {
	Name string
}

// ClientUpdate é o pacote de atualização enviado pelo cliente.
type ClientUpdate struct {
	PlayerName     string 
	SequenceNumber int64
	NewPlayerState PlayerState
	TileChanged    *MapTileChange
}

type MapTileChange struct {
	X, Y       int
	NewElement ElementoDoMapa
}

// ta vazio aqui pq tenho que passar algo nas funções e é isso.
type EmptyArgs struct{}
