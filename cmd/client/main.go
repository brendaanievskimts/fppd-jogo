package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/nsf/termbox-go"
)

// JogoCliente guarda o estado local do jogo, que é uma cópia do estado do servidor.
// O Mutex é crucial para evitar que a gente tente desenhar o estado enquanto ele está sendo atualizado.
type JogoCliente struct {
	Mutex sync.Mutex
	State GameState
}
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

// ---------------------------------------------------------------------------------
// LÓGICA DE INTERFACE (ADAPTADA DO SEU CÓDIGO ORIGINAL)
// ---------------------------------------------------------------------------------

// EventoTeclado representa uma ação do teclado
type EventoTeclado struct {
	Tipo  string // "sair", "mover"
	Tecla rune
}

// interfaceIniciar inicializa a biblioteca termbox
func interfaceIniciar() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
}

// interfaceFinalizar fecha a biblioteca termbox
func interfaceFinalizar() {
	termbox.Close()
}

// interfaceLerEventoTeclado lê uma tecla e a transforma em uma ação do jogo
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
	return EventoTeclado{} // Ignora outras teclas/eventos
}

// interfaceDesenharJogo é a função principal de renderização.
// Foi adaptada para receber o JogoCliente.
func interfaceDesenharJogo(jogo *JogoCliente) {
	jogo.Mutex.Lock()         // Trava para ler o estado com segurança
	defer jogo.Mutex.Unlock() // Libera a trava no final

	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	// --- A LÓGICA MULTIPLAYER ---
	// 1. Desenha um '☺' para CADA jogador na lista que recebemos do servidor
	for _, player := range jogo.State.Players {
		termbox.SetCell(player.X, player.Y, '☺', termbox.ColorWhite, termbox.ColorDefault)
	}

	// 2. Desenha informações na tela
	msg := fmt.Sprintf("Jogadores online: %d. Use WASD para mover e ESC para sair.", len(jogo.State.Players))
	for i, c := range msg {
		termbox.SetCell(i, 0, c, termbox.ColorWhite, termbox.ColorDefault)
	}

	termbox.Flush() // Envia tudo para o terminal
}


// ---------------------------------------------------------------------------------
// LÓGICA PRINCIPAL DO CLIENTE (main)
// ---------------------------------------------------------------------------------

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Uso: go run . <ip_do_servidor> <seu_nome>")
		return
	}
	serverIP := os.Args[1]
	playerID := os.Args[2]
	porta := 1234

	// 1. CONECTA AO SERVIDOR
	client, err := rpc.Dial("tcp", fmt.Sprintf("%s:%d", serverIP, porta))
	if err != nil {
		log.Fatal("Erro ao conectar:", err)
	}
	defer client.Close()

	// 2. ENTRA NO JOGO
	var gameState GameState
	err = client.Call("GameServer.JoinGame", playerID, &gameState)
	if err != nil {
		log.Fatal("Erro ao entrar no jogo:", err)
	}

	// 3. INICIA A INTERFACE
	interfaceIniciar()
	defer interfaceFinalizar()

	// Cria a variável local do jogo
	jogoLocal := &JogoCliente{
		State: gameState,
	}

	// 4. GOROUTINE PARA RECEBER ATUALIZAÇÕES (POLLING)
	// Esta é a thread que fica buscando o estado atual do jogo
	go func() {
		for {
			var latestState GameState
			// Pede ao servidor: "Me dê o estado mais recente do jogo"
			err := client.Call("GameServer.GetGameState", 0, &latestState)
			if err == nil {
				// Se receber, atualiza o estado local
				jogoLocal.Mutex.Lock()
				jogoLocal.State = latestState
				jogoLocal.Mutex.Unlock()

				// E redesenha a tela com as novas informações
				interfaceDesenharJogo(jogoLocal)
			}
			// Espera um pouco antes de pedir de novo
			time.Sleep(100 * time.Millisecond)
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
				case 'w': dy = -1
				case 'a': dx = -1
				case 's': dy = 1
				case 'd': dx = 1
			}

			seqNum++ // Novo comando, novo número
			cmd := ClientCommand{
				ClientID:       playerID,
				SequenceNumber: seqNum,
				Action:         "move",
				Params:         map[string]interface{}{"dx": dx, "dy": dy},
			}

			var reply bool
			// Envia o comando para o servidor e não se preocupa com o resultado imediato
			// A atualização da posição virá pela goroutine de polling
			client.Call("GameServer.SendCommand", cmd, &reply)
		}
	}
}