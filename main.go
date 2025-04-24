// main.go - Loop principal do jogo
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)
//as duas próximas funções são pra tentar redimensionar o terminal de acordo com o tamanho do mapa
//acho que não tão dando certo, elas fazem um script pra rodar isso
func redimensionarTerminal(linhas, colunas int) {
	// Verifica o sistema operacional
	switch runtime.GOOS {
	case "windows":
		// Redimensiona o terminal no Windows usando a sequência ANSI
		os.Stdout.WriteString(fmt.Sprintf("\x1b[8;%d;%dt", linhas, colunas))
	case "linux", "darwin": // Linux e macOS
		// Redimensiona o terminal no Linux/macOS com o comando resize
		cmd := exec.Command("resize", "-s", fmt.Sprintf("%d", linhas), fmt.Sprintf("%d", colunas))
		cmd.Run() // Executa o comando para redimensionar o terminal
	default:
		fmt.Println("Sistema operacional não suportado para redimensionamento automático.")
	}
}

// Função para verificar e redimensionar o terminal de acordo com o mapa
func verificarERedimensionar(mapa [][]Elemento) {
	// Obtém o número de linhas e colunas do mapa
	linhas := len(mapa)
	colunas := len(mapa[0])

	// Redimensiona o terminal para o tamanho do mapa
	redimensionarTerminal(linhas, colunas)
}

func main() {
		// Carrega o mapa
		jogo := jogoNovo()
		if err := jogoCarregarMapa("mapa.txt", &jogo); err != nil {
			panic(err)
		}
	
		// Redimensiona o terminal com base nas dimensões do mapa (somente uma vez)
		verificarERedimensionar(jogo.Mapa)
	
		// Inicia o jogo
		interfaceIniciar()
		defer interfaceFinalizar()

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

	atualizarTela := make(chan bool)

	// Goroutine para redesenhar a tela periodicamente
	go func() {
		for {
			<-atualizarTela // Espera pelo sinal de que é hora de desenhar
			interfaceDesenharJogo(&jogo) // Atualiza a tela
		}
	}()

	for i := range jogo.Aliens {
		alien := &jogo.Aliens[i]
		go func(alien *AlienMovel) {
			for {
				select {
				case <-done:
					return
				default:
					moverAlien(alien, &jogo)
					atualizarTela <- true
					time.Sleep(300 * time.Millisecond)
				}
			}
		}(alien)
	}
	
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


}
