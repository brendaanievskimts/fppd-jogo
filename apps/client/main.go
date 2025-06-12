package main

import (
	"fppd-jogo/game"
	"fppd-jogo/logica_jogo"
	"log"
	"net/rpc"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Uso: go run client/client.go <ip_do_servidor:porta> <SeuNomeDeJogador>")
		log.Fatalf("Exemplo: go run client/client.go localhost:1234 Professor_Marcelo")
	}
	serverAddress := os.Args[1]
	playerName := os.Args[2]

	client, err := rpc.Dial("tcp", serverAddress)
	if err != nil {
		log.Fatalf("Erro ao conectar no servidor em %s: %v", serverAddress, err)
	}
	defer client.Close()

	Iniciar()
	defer Finalizar()

	var estadoDoServidor game.GameState
	joinReq := game.JoinRequest{Name: playerName}
	err = client.Call("GameServer.JoinGame", joinReq, &estadoDoServidor)
	if err != nil {
		log.Fatal("Erro ao entrar no jogo: ", err)
	}

	jogoLocal := logica_jogo.NovoJogo(playerName)
	traduzirParaJogoLocal(estadoDoServidor, jogoLocal)

	eventQueue := make(chan EventoTeclado)
	go func() {
		for {
			eventQueue <- LerEventoTeclado()
		}
	}()

	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	var sequenceNumber int64 = 0

	for {
		DesenharJogo(jogoLocal)

		select {
		case ev := <-eventQueue:
			estadoJogadorAntes, exists := jogoLocal.Players[playerName]
			if !exists {
				continue
			}

			if !jogoLocal.ExecutarAcao(ev) {
				return
			}

			estadoJogadorDepois := jogoLocal.Players[playerName]

			update, mudou := criarUpdateParaServidor(estadoJogadorAntes, estadoJogadorDepois, playerName, &sequenceNumber)
			if mudou {
				var success bool
				go client.Call("GameServer.UpdateState", update, &success)
			}

		case <-ticker.C:
			err := client.Call("GameServer.GetGameState", &game.EmptyArgs{}, &estadoDoServidor)
			if err == nil {
				traduzirParaJogoLocal(estadoDoServidor, jogoLocal)
			}
		}
	}
}

func traduzirParaJogoLocal(gs game.GameState, jogoLocal *logica_jogo.Jogo) {
	mapaLocal := make([][]logica_jogo.Elemento, len(gs.Mapa))
	for y, linha := range gs.Mapa {
		mapaLocal[y] = make([]logica_jogo.Elemento, len(linha))
		for x, elemServidor := range linha {
			switch elemServidor.Simbolo {
			case '▤':
				mapaLocal[y][x] = logica_jogo.Parede
			case '♣':
				mapaLocal[y][x] = logica_jogo.Vegetacao
			default:
				mapaLocal[y][x] = logica_jogo.Vazio
			}
		}
	}
	jogoLocal.Mapa = mapaLocal
	jogoLocal.Players = gs.Players
	jogoLocal.StatusMsg = gs.Status
}

func criarUpdateParaServidor(antes, depois game.PlayerState, myName string, seq *int64) (game.ClientUpdate, bool) {
	if antes.X == depois.X && antes.Y == depois.Y && antes.VegetacoesColetadas == depois.VegetacoesColetadas {
		return game.ClientUpdate{}, false
	}

	*seq++

	var tileChange *game.MapTileChange
	if antes.VegetacoesColetadas < depois.VegetacoesColetadas {
		tileChange = &game.MapTileChange{
			X:          depois.X,
			Y:          depois.Y,
			NewElement: game.ElementoDoMapa{Simbolo: ' ', Tangivel: false},
		}
	}

	update := game.ClientUpdate{
		PlayerName:     myName,
		SequenceNumber: *seq,
		NewPlayerState: depois,
		TileChanged:    tileChange,
	}
	return update, true
}
