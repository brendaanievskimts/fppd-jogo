package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"sync"
	"time"

	"fppd-jogo/game" // <-- Importa os tipos compartilhados (use o nome do seu módulo)
	"github.com/nsf/termbox-go"
)

// JogoCliente guarda o estado local do jogo, que é uma cópia do estado do servidor.
type JogoCliente struct {
	Mutex sync.Mutex
	State game.GameState
}

// EventoTeclado representa uma ação do teclado
type EventoTeclado struct {
	Tipo  string // "sair", "mover"
	Tecla rune
}

func interfaceIniciar() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
}

func interfaceFinalizar() {
	termbox.Close()
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

func interfaceDesenharJogo(jogo *JogoCliente, selfID string) {
	jogo.Mutex.Lock()
	defer jogo.Mutex.Unlock()

	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	// Desenha cada jogador na sua posição
	for id, player := range jogo.State.Players {
		char := '☺'
		color := termbox.ColorWhite
		// Destaca o nosso jogador com uma cor diferente
		if id == selfID {
			color = termbox.ColorYellow
		}
		termbox.SetCell(player.X, player.Y, char, color, termbox.ColorDefault)
	}

	// Desenha informações na tela
	msg := fmt.Sprintf("Jogadores online: %d. Use WASD para mover e ESC para sair.", len(jogo.State.Players))
	for i, c := range msg {
		termbox.SetCell(i, 0, c, termbox.ColorWhite, termbox.ColorDefault)
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
	porta := 1234

	// 1. CONECTA AO SERVIDOR
	log.Printf("Tentando conectar a %s:%d...", serverIP, porta)
	client, err := rpc.Dial("tcp", fmt.Sprintf("%s:%d", serverIP, porta))
	if err != nil {
		log.Fatalf("ERRO AO CONECTAR: %v. Verifique se o IP está correto e o servidor está rodando.", err)
	}
	defer client.Close()
	log.Println("Conectado ao servidor com sucesso!")

	// 2. ENTRA NO JOGO
	var gameState game.GameState
	log.Printf("Enviando pedido para entrar no jogo como '%s'...", playerID)
	err = client.Call("GameServer.JoinGame", playerID, &gameState)
	if err != nil {
		log.Fatalf("ERRO AO ENTRAR NO JOGO: %v", err)
	}
	log.Println("Você entrou no jogo! Estado inicial recebido.")

	// 3. INICIA A INTERFACE E O JOGO LOCAL
	interfaceIniciar()
	defer interfaceFinalizar()

	jogoLocal := &JogoCliente{
		State: gameState,
	}
	interfaceDesenharJogo(jogoLocal, playerID) // Desenha a tela inicial

	// 4. GOROUTINE PARA RECEBER ATUALIZAÇÕES (POLLING)
	go func() {
		for {
			var latestState game.GameState
			// CORREÇÃO: Usa a struct vazia para a chamada RPC, como definido no servidor.
			err := client.Call("GameServer.GetGameState", &game.EmptyArgs{}, &latestState)
			if err != nil {
				// Se a conexão cair, o cliente vai fechar
				log.Printf("Erro ao buscar estado do jogo: %v. A conexão pode ter caído.", err)
				time.Sleep(2 * time.Second) // Evita spam de logs
				continue
			}

			jogoLocal.Mutex.Lock()
			jogoLocal.State = latestState
			jogoLocal.Mutex.Unlock()

			interfaceDesenharJogo(jogoLocal, playerID) // Redesenha a tela com as novas informações
			
			time.Sleep(100 * time.Millisecond) // Espera um pouco antes de pedir de novo
		}
	}()

	// 5. LOOP PRINCIPAL PARA LER TECLADO E ENVIAR COMANDOS
	var seqNum int64 = 0
	for {
		evento := interfaceLerEventoTeclado()

		if evento.Tipo == "sair" {
			break
		}

		if evento.Tipo == "move" {
			dx, dy := 0, 0
			switch evento.Tecla {
			case 'w':
				dy = -1
			case 'a':
				dx = -1
			case 's':
				dy = 1
			case 'd':
				dx = 1
			}

			if dx != 0 || dy != 0 {
				seqNum++ // Novo comando, novo número
				cmd := game.ClientCommand{
					ClientID:       playerID,
					SequenceNumber: seqNum,
					Action:         "move",
					Params:         map[string]interface{}{"dx": dx, "dy": dy},
				}

				var reply bool
				// Envia o comando para o servidor de forma assíncrona
				client.Go("GameServer.SendCommand", cmd, &reply, nil)
			}
		}
	}
}
