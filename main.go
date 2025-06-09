// main.go - Loop principal do jogo
package main

import (
	"os"
	"time"
)

func main() {
	jogo := jogoNovo()
	if err := jogoCarregarMapa("mapa.txt", &jogo); err != nil {
		panic(err)
	}

	// Inicializa a interface (termbox)
	interfaceIniciar()
	defer interfaceFinalizar()

	// Canais
	posicaoJogador := make(chan [2]int, 10) // Buffer para 10 posições
	done := make(chan struct{})
	ativarArmadilha := make(chan bool, 1) // Buffer para 1 sinal
	jogo.VegChan = make(chan int, 10)

	// Inicia goroutines
	//if jogo.Inimigos != nil {
	//	go inimigoPatrulhar(&jogo, posicaoJogador, done) // Apenas UMA goroutine para o inimigo
	//}
	go armadilha(&jogo, ativarArmadilha, done)
	go timerJogo(&jogo, done)
	atualizarTela := make(chan bool)

	// Desenha o estado inicial do jogo
	interfaceDesenharJogo(&jogo)

	// Goroutine para redesenhar a tela periodicamente
	go func() {
		for {
			<-atualizarTela              // Espera pelo sinal de que é hora de desenhar
			interfaceDesenharJogo(&jogo) // Atualiza a tela
		}
	}()

	for {
		jogo.Mutex.Lock()
		gameOver := jogo.GameOver
		jogo.Mutex.Unlock()

		if gameOver {
			// Feche o canal done aqui, uma única vez
			close(done)
			time.Sleep(3 * time.Second)
			break
		}
		evento := interfaceLerEventoTeclado()
		if !personagemExecutarAcao(evento, &jogo) {
			close(done)
			break
		}

		// Envia posição apenas se o jogador se moveu
		if evento.Tipo == "mover" {
			select {
			case posicaoJogador <- [2]int{jogo.PosX, jogo.PosY}: // Não bloqueante
			default: // Descarta posição se o canal estiver cheio
			}
		}

		// Verifica armadilhas (não-bloqueante)
		select {
		case ativarArmadilha <- jogadorPertoDeArmadilha(&jogo, jogo.PosX, jogo.PosY):
		default:
		}

		interfaceDesenharJogo(&jogo)
	}
	os.Exit(0) // Encerra o programa

	// Loop principal de entrada
	for {
		evento := interfaceLerEventoTeclado()
		if continuar := personagemExecutarAcao(evento, &jogo); !continuar {
			break
		}
		interfaceDesenharJogo(&jogo)
	}
}
