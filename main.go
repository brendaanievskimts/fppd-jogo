// main.go - Loop principal do jogo
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	interfaceIniciar()
	defer interfaceFinalizar()

	mapaFile := "mapa.txt"
	if len(os.Args) > 1 {
		mapaFile = os.Args[1]
	}

	jogo := jogoNovo()
	jogo.Vidas = 3
	jogo.GameOver = false
	if err := jogoCarregarMapa(mapaFile, &jogo); err != nil {
		panic(err)
	}

	interfaceDesenharJogo(&jogo)

	posicaoJogador := make(chan [2]int, 10)
	done := make(chan struct{})
	ativarArmadilha := make(chan bool, 10)

	for _, inimigo := range jogo.Inimigos {
		go inimigoPatrulhar(&jogo, posicaoJogador, done, inimigo)
	}

	go itemCura(&jogo, done)
	go armadilha(&jogo, ativarArmadilha, done)

	ticker := time.NewTicker(100 * time.Millisecond)

	for {
		select {
		case <-ticker.C:
			fmt.Println("Ticker: Enviando posição do jogador")
			select {
			case posicaoJogador <- [2]int{jogo.PosX, jogo.PosY}:
			default:
			}
			fmt.Println("Ticker: Verificando armadilha")
			select {
			case ativarArmadilha <- jogadorPertoDeArmadilha(&jogo, jogo.PosX, jogo.PosY):
			default:
			}
			interfaceDesenharJogo(&jogo)

		default:
			fmt.Println("Lendo evento do teclado")
			evento := interfaceLerEventoTeclado()
			fmt.Println("Evento lido:", evento.Tipo, evento.Tecla)
			if evento.Tipo == "" {
				continue
			}

			if jogo.GameOver && evento.Tipo != "sair" {
				// No GameOver, ainda permite interagir para mostrar mensagens
				if evento.Tipo == "interagir" {
					personagemInteragir(&jogo)
				}
				interfaceDesenharJogo(&jogo)
				continue
			}

			if !personagemExecutarAcao(evento, &jogo) {
				close(done)
				close(ativarArmadilha)
				return
			}

			if evento.Tipo == "mover" {
				select {
				case posicaoJogador <- [2]int{jogo.PosX, jogo.PosY}:
				default:
				}
			}

			interfaceDesenharJogo(&jogo)
		}
	}
}
