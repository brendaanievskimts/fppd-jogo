package main

import (
	"bufio"
	"fppd-jogo/game"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
)

type GameServer struct {
	mutex         sync.Mutex
	gameState     game.GameState
	lastProcessed map[string]int64
	spawnPoints   [][2]int 
}

func (s *GameServer) preencherSpawnPoints() {
	for y, linha := range s.gameState.Mapa {
		for x, elem := range linha {
			// Um ponto de spawn é qualquer lugar que não seja uma parede.
			if !elem.Tangivel {
				s.spawnPoints = append(s.spawnPoints, [2]int{x, y})
			}
		}
	}
}

// JoinGame: Jogador entra no jogo e recebe uma posição aleatória.
func (s *GameServer) JoinGame(request game.JoinRequest, reply *game.GameState) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.gameState.Players[request.Name]; !exists {
		if len(s.spawnPoints) == 0 {
			return fmt.Errorf("nenhum ponto de spawn válido no mapa")
		}
		spawnPoint := s.spawnPoints[rand.Intn(len(s.spawnPoints))]

		s.gameState.Players[request.Name] = game.PlayerState{
			Name:                request.Name,
			X:                   spawnPoint[0],
			Y:                   spawnPoint[1],
			Vida:                3,
			VegetacoesColetadas: 0,
		}
		log.Printf("Jogador '%s' entrou no jogo na posição (%d, %d).", request.Name, spawnPoint[0], spawnPoint[1])
	} else {
		log.Printf("Jogador '%s' reconectou.", request.Name)
	}
	*reply = s.gameState
	return nil
}

func (s *GameServer) GetGameState(args *game.EmptyArgs, reply *game.GameState) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	*reply = s.gameState
	return nil
}

func (s *GameServer) UpdateState(update game.ClientUpdate, reply *bool) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.gameState.Players[update.PlayerName]; !exists {
		log.Printf("Rejeitando atualização do jogador desconhecido: %s", update.PlayerName)
		*reply = false
		return fmt.Errorf("jogador '%s' não existe no jogo", update.PlayerName)
	}

	if lastSeq, found := s.lastProcessed[update.PlayerName]; found && update.SequenceNumber <= lastSeq {
		*reply = true
		return nil 
	}

	s.gameState.Players[update.PlayerName] = update.NewPlayerState
	if update.TileChanged != nil {
		tc := update.TileChanged
		if tc.Y >= 0 && tc.Y < len(s.gameState.Mapa) && tc.X >= 0 && tc.X < len(s.gameState.Mapa[tc.Y]) {
			s.gameState.Mapa[tc.Y][tc.X] = tc.NewElement
		}
	}
	s.lastProcessed[update.PlayerName] = update.SequenceNumber
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
		var linhaElems []game.ElementoDoMapa
		for _, ch := range linha {
			var elem game.ElementoDoMapa
			switch ch {
			case '▤':
				elem = game.ElementoDoMapa{Simbolo: '▤', Tangivel: true}
			case '♣':
				elem = game.ElementoDoMapa{Simbolo: '♣', Tangivel: false}
			default:
				elem = game.ElementoDoMapa{Simbolo: ' ', Tangivel: false}
			}
			linhaElems = append(linhaElems, elem)
		}
		gameState.Mapa = append(gameState.Mapa, linhaElems)
	}
}

func main() {
	servidor := &GameServer{
		gameState: game.GameState{
			Players: make(map[string]game.PlayerState),
			Mapa:    [][]game.ElementoDoMapa{},
			Status:  "Servidor online. Bem-vindo!",
		},
		lastProcessed: make(map[string]int64),
		spawnPoints:   make([][2]int, 0),
	}

	carregarMapa("mapa.txt", &servidor.gameState)
	servidor.preencherSpawnPoints()

	rpc.Register(servidor)
	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal("Erro ao escutar: ", err)
	}
	defer listener.Close()

	log.Println("Servidor RPC esperando chamadas na porta 1234...")
	rpc.Accept(listener)
}
