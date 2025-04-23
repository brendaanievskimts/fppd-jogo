// main.go - Loop principal do jogo
package main

import "time"

func main() {
	interfaceIniciar()
	defer interfaceFinalizar()

	// Carrega mapa
	jogo := jogoNovo()
	if err := jogoCarregarMapa("mapa.txt", &jogo); err != nil {
		panic(err)
	}

	// Canais
	posicaoJogador := make(chan [2]int, 10) // Buffer para 10 posições
	done := make(chan struct{})
	ativarArmadilha := make(chan bool, 1) // Buffer para 1 sinal
	jogo.VegChan = make(chan int, 10)

	// Inicia goroutines
	for _, inimigo := range jogo.Inimigos {
		go inimigoPatrulhar(&jogo, posicaoJogador, done, inimigo)
	}
	go armadilha(&jogo, ativarArmadilha, done)
	go timerJogo(&jogo, done)

	// No loop de desenho, adicione:
	//for _, inimigo := range jogo.Inimigos {
	//	if inimigo.Ativo {
	//		interfaceDesenharElemento(inimigo.X, inimigo.Y, Inimigo)
	//	}
	//}
	//go armadilha(&jogo, ativarArmadilha, done)

	// Loop principal
	for {
		jogo.Mutex.Lock()
		gameOver := jogo.GameOver
		jogo.Mutex.Unlock()

		if gameOver {
			time.Sleep(3 * time.Second) // Tempo para ler a mensagem final
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
}
