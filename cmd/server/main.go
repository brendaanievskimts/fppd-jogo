package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
)

// GameServer é o objeto principal que gerenciará nosso jogo.
type GameServer struct {
	mutex         sync.Mutex                 // Para proteger o acesso aos dados de múltiplas conexões
	gameState     GameState           // Guarda o estado atual de TODOS os jogadores
	lastProcessed map[string]int64           // Guarda o último comando processado de cada jogador (para idempotência)
}
type PlayerState struct {
	ID   string
	X, Y int
}

// O estado COMPLETO do jogo, com todos os jogadores.
// É isso que o servidor enviará para os clientes.
type GameState struct {
	Players map[string]PlayerState
}

// O comando que o cliente envia para o servidor.
type ClientCommand struct {
	ClientID       string                 // Para o servidor saber quem enviou o comando
	SequenceNumber int64                  // Para garantir a idempotência (execução única)
	Action         string                 // "move", "interact", etc.
	Params         map[string]interface{} // Dados extras, como a direção do movimento
}

// (Continuação do cmd/server/main.go)

// JoinGame é um método RPC que um cliente chama para entrar no jogo.
func (s *GameServer) JoinGame(playerID string, reply *GameState) error {
	s.mutex.Lock()         // Trava para garantir que apenas uma thread modifique os dados por vez
	defer s.mutex.Unlock() // Garante que a trava será liberada ao final da função

	// Adiciona o novo jogador em uma posição inicial (ex: 5,5)
	s.gameState.Players[playerID] = PlayerState{ID: playerID, X: 5, Y: 5}
	log.Printf("Jogador '%s' entrou no jogo.", playerID)

	// Retorna o estado completo do jogo para o novo jogador
	*reply = s.gameState
	return nil
}

// GetGameState é o método que o cliente chama periodicamente (polling) para obter o estado mais recente.
func (s *GameServer) GetGameState(_, reply *GameState) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Simplesmente retorna uma cópia do estado atual
	*reply = s.gameState
	return nil
}

// SendCommand é o método que processa as ações dos jogadores (como mover).
func (s *GameServer) SendCommand(cmd ClientCommand, reply *bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// --- LÓGICA DE IDEMPOTÊNCIA (Requisito "exactly-once") ---
	lastSeq, found := s.lastProcessed[cmd.ClientID]
	if found && cmd.SequenceNumber <= lastSeq {
		// Se já processamos este comando, ignoramos a lógica mas retornamos sucesso.
		*reply = true
		return nil
	}

	// Processa a ação do comando se for nova
	if cmd.Action == "move" {
		if player, ok := s.gameState.Players[cmd.ClientID]; ok {
			// RPC converte números genéricos para float64, então precisamos converter de volta para int
			dx := int(cmd.Params["dx"].(float64))
			dy := int(cmd.Params["dy"].(float64))
			
			// Atualiza a posição do jogador
			player.X += dx
			player.Y += dy
			s.gameState.Players[cmd.ClientID] = player
		}
	}

	// Atualiza a "memória" do servidor com o número do último comando processado
	s.lastProcessed[cmd.ClientID] = cmd.SequenceNumber
	*reply = true // Informa ao cliente que o comando foi aceito
	return nil
}

// A função main é o ponto de partida do nosso programa.
func main() {
	porta := 1234

	// Cria uma nova instância do nosso servidor de jogo
	servidor := &GameServer{
		gameState: GameState{
			Players: make(map[string]PlayerState),
		},
		lastProcessed: make(map[string]int64),
	}

	// Registra o servidor para que ele possa ser encontrado pela rede via RPC
	rpc.Register(servidor)

	// Começa a "ouvir" por conexões de rede na porta especificada
	// "0.0.0.0" significa que ele aceitará conexões de qualquer computador
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", porta))
	if err != nil {
		log.Fatal("Erro ao iniciar o servidor: ", err)
	}
	defer listener.Close()

	log.Printf("Servidor aguardando conexões na porta %d...", porta)

	// Aceita conexões para sempre
	rpc.Accept(listener)
}