// main.go - Loop principal do jogo
package main

func main() {
	interfaceIniciar()
	defer interfaceFinalizar()

	// Carrega mapa
	jogo := jogoNovo()
	if err := jogoCarregarMapa("mapa.txt", &jogo); err != nil {
		panic(err)
	}

	// Canais com buffer para evitar bloqueio imediato
	posicaoJogador := make(chan [2]int, 10) // Buffer para 10 posições
	done := make(chan struct{})
	ativarArmadilha := make(chan bool, 1) // Buffer para 1 sinal

	// Inicia inimigos
	for _, inimigo := range jogo.Inimigos {
		go inimigoPatrulhar(&jogo, posicaoJogador, done, inimigo)
	}

	// No loop de desenho, adicione:
	for _, inimigo := range jogo.Inimigos {
		if inimigo.Ativo {
			interfaceDesenharElemento(inimigo.X, inimigo.Y, Inimigo)
		}
	}
	go armadilha(&jogo, ativarArmadilha, done)

	// Loop principal
	for {
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
