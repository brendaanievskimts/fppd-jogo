package main

import (
	"bufio"
	"fppd-jogo/game"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
)

// Elementos visuais (apenas para o servidor usar como referência)
var (
	Parede    = game.Elemento{'▤', true}
	Vegetacao = game.Elemento{'♣', false}
	Vazio     = game.Elemento{' ', false}
)

type GameServer struct {
	mutex         sync.Mutex
	gameState     game.GameState
	lastProcessed map[string]int64
}

// JoinGame: Jogador entra no jogo
func (s *GameServer) JoinGame(playerID string, reply *game.GameState) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.gameState.Players[playerID]; !exists {
		s.gameState.Players[playerID] = game.PlayerState{ID: playerID, X: 9, Y: 2, GramasComidas: 0}
		log.Printf("Jogador '%s' entrou no jogo.", playerID)
	} else {
		log.Printf("Jogador '%s' reconectou.", playerID)
	}
	*reply = s.gameState
	return nil
}

// GetGameState: Cliente pede o estado atual
func (s *GameServer) GetGameState(args *game.EmptyArgs, reply *game.GameState) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	*reply = s.gameState
	return nil
}

// SendCommand: Processa o movimento do jogador e a lógica de "comer"
func (s *GameServer) SendCommand(cmd game.ClientCommand, reply *bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Lógica de "exactly-once"
	if lastSeq, found := s.lastProcessed[cmd.ClientID]; found && cmd.SequenceNumber <= lastSeq {
		*reply = true
		return nil
	}

	player, ok := s.gameState.Players[cmd.ClientID]
	if !ok {
		*reply = false
		return fmt.Errorf("jogador não encontrado")
	}

	if cmd.Action == "move" {
		dx := int(cmd.Params["dx"].(int))
		dy := int(cmd.Params["dy"].(int))

		nx, ny := player.X+dx, player.Y+dy

		// Verifica se pode mover (não é parede)
		if nx >= 0 && ny >= 0 && ny < len(s.gameState.Mapa) && nx < len(s.gameState.Mapa[ny]) && !s.gameState.Mapa[ny][nx].Tangivel {
			// Se o destino tem grama, come!
			if s.gameState.Mapa[ny][nx].Simbolo == Vegetacao.Simbolo {
				player.GramasComidas++
				s.gameState.Mapa[ny][nx] = Vazio // Remove a grama
			}
			player.X = nx
			player.Y = ny
			s.gameState.Players[cmd.ClientID] = player
		}
	}

	s.lastProcessed[cmd.ClientID] = cmd.SequenceNumber
	*reply = true
	return nil
}

func carregarMapa(nome string, gameState *game.GameState) {
	arq, err := os.Open(nome)
	if err != nil {
		panic(err)
	}
	defer arq.Close()
	scanner := bufio.NewScanner(arq)
	for scanner.Scan() {
		linha := scanner.Text()
		var linhaElems []game.Elemento
		for _, ch := range linha {
			elem := Vazio
			switch ch {
			case Parede.Simbolo:
				elem = Parede
			case Vegetacao.Simbolo:
				elem = Vegetacao
			}
			linhaElems = append(linhaElems, elem)
		}
		gameState.Mapa = append(gameState.Mapa, linhaElems)
	}
}

// Goroutine para o timer do jogo
func gameTimer(s *GameServer) {
	tempoTotal := 60
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for i := tempoTotal; i > 0; i-- {
		s.mutex.Lock()
		s.gameState.Status = fmt.Sprintf("Tempo restante: %d segundos", i)
		s.mutex.Unlock()
		<-ticker.C
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	vencedorID := ""
	maiorPontuacao := -1
	for id, player := range s.gameState.Players {
		if player.GramasComidas > maiorPontuacao {
			maiorPontuacao = player.GramasComidas
			vencedorID = id
		}
	}
	s.gameState.Status = fmt.Sprintf("FIM DE JOGO! Vencedor: %s com %d gramas!", vencedorID, maiorPontuacao)
}

func main() {
	servidor := &GameServer{
		gameState: game.GameState{
			Players: make(map[string]game.PlayerState),
			Mapa:    [][]game.Elemento{},
		},
		lastProcessed: make(map[string]int64),
	}

	carregarMapa("mapa.txt", &servidor.gameState) // Carrega o mapa no estado do servidor
	go gameTimer(servidor)                      // Inicia o timer do jogo

	rpc.Register(servidor)
	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal("Erro no listener: ", err)
	}
	defer listener.Close()

	log.Println("Servidor RPC pronto na porta 1234...")
	rpc.Accept(listener)
}