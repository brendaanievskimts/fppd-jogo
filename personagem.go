// personagem.go - Funções para movimentação e ações do personagem
package main

import (
	"fmt"
)

func personagemMover(tecla rune, jogo *Jogo) {
	dx, dy := 0, 0
	switch tecla {
	case 'w':
		dy = -1 // Move para cima
	case 'a':
		dx = -1 // Move para a esquerda
	case 's':
		dy = 1 // Move para baixo
	case 'd':
		dx = 1 // Move para a direita
	}

	nx, ny := jogo.PosX+dx, jogo.PosY+dy
	verificaColisaoVegetacao(jogo, nx, ny)
	if jogoPodeMoverPara(jogo, nx, ny) {
		jogoMoverElemento(jogo, jogo.PosX, jogo.PosY, dx, dy)
		jogo.PosX, jogo.PosY = nx, ny
	}
}
func personagemInteragir(jogo *Jogo) {
	jogo.StatusMsg = fmt.Sprintf("Interagindo em (%d, %d)", jogo.PosX, jogo.PosY)
}

// Processa o evento do teclado e executa a ação correspondente
func personagemExecutarAcao(ev EventoTeclado, jogo *Jogo) bool {
	switch ev.Tipo {
	case "sair":
		// Retorna false para indicar que o jogo deve terminar
		return false
	case "interagir":
		// Executa a ação de interação
		personagemInteragir(jogo)
	case "mover":
		// Move o personagem com base na tecla
		personagemMover(ev.Tecla, jogo)
	}
	return true // Continua o jogo
}

func verificaColisaoVegetacao(jogo *Jogo, nx, ny int) {
	if jogo.Mapa[ny][nx].simbolo == Vegetacao.simbolo {
		jogo.Mutex.Lock()
		defer jogo.Mutex.Unlock()

		jogo.Mapa[ny][nx] = Vazio
		jogo.VegetacoesColetadas++

		select {
		case jogo.VegChan <- jogo.VegetacoesColetadas:
		default:
		}
	}
}