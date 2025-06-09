package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
	"fppd-jogo/game" 
)

// GameServer é o objeto principal que gerenciará nosso jogo.
type GameServer struct {
	mutex         sync.Mutex
	gameState     game.GameState
	lastProcessed map[string]int64
}

// JoinGame é um método RPC que um cliente chama para entrar no jogo.
func (s *GameServer) JoinGame(playerID string, reply *game.GameState) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Adiciona o novo jogador em uma posição inicial (ex: 5,5)
	if _, exists := s.gameState.Players[playerID]; !exists {
		s.gameState.Players[playerID] = game.PlayerState{ID: playerID, X: 5, Y: 5}
		log.Printf("Jogador '%s' entrou no jogo.", playerID)
	} else {
		log.Printf("Jogador '%s' reconectou.", playerID)
	}

	// Retorna o estado completo do jogo para o novo jogador
	*reply = s.gameState
	return nil
}

// GetGameState é o método que o cliente chama periodicamente para obter o estado mais recente.
// A assinatura agora usa `*game.EmptyArgs` para ser explícita.
func (s *GameServer) GetGameState(args *game.EmptyArgs, reply *game.GameState) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Simplesmente retorna uma cópia do estado atual
	*reply = s.gameState
	return nil
}

// SendCommand é o método que processa as ações dos jogadores (como mover).
func (s *GameServer) SendCommand(cmd game.ClientCommand, reply *bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	lastSeq, found := s.lastProcessed[cmd.ClientID]
	if found && cmd.SequenceNumber <= lastSeq {
		*reply = true
		return nil
	}

	if cmd.Action == "move" {
		if player, ok := s.gameState.Players[cmd.ClientID]; ok {
			dx := int(cmd.Params["dx"].(int64))
			dy := int(cmd.Params["dy"].(int64))
			player.X += dx
			player.Y += dy
			s.gameState.Players[cmd.ClientID] = player
		}
	}

	s.lastProcessed[cmd.ClientID] = cmd.SequenceNumber
	*reply = true
	return nil
}

func main() {
	porta := 1234

	servidor := &GameServer{
		gameState: game.GameState{
			Players: make(map[string]game.PlayerState),
		},
		lastProcessed: make(map[string]int64),
	}

	rpc.Register(servidor)
	log.Println("Servidor RPC registrado.")

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", porta))
	if err != nil {
		log.Fatal("Erro ao iniciar o listener: ", err)
	}
	defer listener.Close()

	log.Printf("Servidor aguardando conexões na porta %d...", porta)
	rpc.Accept(listener)
}
