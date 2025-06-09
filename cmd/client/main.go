package main

import (
	"fppd-jogo/game" // Use o nome do seu módulo
	"fmt"
	"log"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

type JogoCliente struct {
	Mutex sync.Mutex
	State game.GameState
}

// ... (as funções de interface e EventoTeclado continuam as mesmas) ...
func interfaceIniciar() {
    if err := termbox.Init(); err != nil {
        panic(err)
    }
}
func interfaceFinalizar() {
    termbox.Close()
}
type EventoTeclado struct {
    Tipo  string
    Tecla rune
}
func interfaceLerEventoTeclado() EventoTeclado {
    ev := termbox.PollEvent()
    if ev.Type == termbox.EventKey {
        if ev.Key == termbox.KeyEsc {
            return EventoTeclado{Tipo: "sair"}
        }
        if ev.Ch == 'w' || ev.Ch == 'a' || ev.Ch == 's' || ev.Ch == 'd' {
            return EventoTeclado{Tipo: "move", Tecla: ev.Ch}
        }
    }
    return EventoTeclado{}
}


// Função de desenho ATUALIZADA
func interfaceDesenharJogo(jogo *JogoCliente, selfID string) {
	jogo.Mutex.Lock()
	defer jogo.Mutex.Unlock()

	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	// 1. Desenha o MAPA primeiro
	for y, linha := range jogo.State.Mapa {
		for x, elem := range linha {
			termbox.SetCell(x, y, elem.Simbolo, termbox.ColorGreen, termbox.ColorDefault)
		}
	}

	// 2. Desenha os JOGADORES por cima do mapa
	for id, player := range jogo.State.Players {
		char := '☺'
		color := termbox.ColorWhite
		if id == selfID {
			color = termbox.ColorYellow // Destaca nosso jogador
		}
		termbox.SetCell(player.X, player.Y, char, color, termbox.ColorDefault)
	}

	// 3. Desenha a BARRA DE STATUS
	nossaPontuacao := 0
	if p, ok := jogo.State.Players[selfID]; ok {
		nossaPontuacao = p.GramasComidas
	}
	statusMsg := fmt.Sprintf("Sua pontuação: %d | %s", nossaPontuacao, jogo.State.Status)
	for i, c := range statusMsg {
		termbox.SetCell(i, len(jogo.State.Mapa)+1, c, termbox.ColorWhite, termbox.ColorDefault)
	}

	termbox.Flush()
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Uso: go run . <ip_do_servidor> <seu_nome>")
		return
	}
	serverIP := os.Args[1]
	playerID := os.Args[2]

	client, err := rpc.Dial("tcp", fmt.Sprintf("%s:%d", serverIP, 1234))
	if err != nil {
		log.Fatalf("ERRO AO CONECTAR: %v", err)
	}
	defer client.Close()

	var gameState game.GameState
	err = client.Call("GameServer.JoinGame", playerID, &gameState)
	if err != nil {
		log.Fatalf("ERRO AO ENTRAR NO JOGO: %v", err)
	}

	interfaceIniciar()
	defer interfaceFinalizar()

	jogoLocal := &JogoCliente{State: gameState}

	// Goroutine para receber atualizações do servidor
	go func() {
		for {
			var latestState game.GameState
			err := client.Call("GameServer.GetGameState", &game.EmptyArgs{}, &latestState)
			if err != nil {
				log.Printf("Erro ao buscar estado: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}
			jogoLocal.Mutex.Lock()
			jogoLocal.State = latestState
			jogoLocal.Mutex.Unlock()
			interfaceDesenharJogo(jogoLocal, playerID) // Redesenha a tela
			time.Sleep(100 * time.Millisecond)
		}
	}()

	var seqNum int64 = 0
	for {
		evento := interfaceLerEventoTeclado()
		if evento.Tipo == "sair" {
			break
		}
		if evento.Tipo == "move" {
			dx, dy := 0, 0
			switch evento.Tecla {
			case 'w': dy = -1
			case 'a': dx = -1
			case 's': dy = 1
			case 'd': dx = 1
			}
			if dx != 0 || dy != 0 {
				seqNum++
				cmd := game.ClientCommand{
					ClientID:       playerID,
					SequenceNumber: seqNum,
					Action:         "move",
					Params:         map[string]interface{}{"dx": dx, "dy": dy},
				}
				var reply bool
				client.Go("GameServer.SendCommand", cmd, &reply, nil)
			}
		}
	}
}