// jogo.go - Funções para manipular os elementos do jogo, como carregar o mapa e mover o personagem
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

// Elemento representa qualquer objeto do mapa (parede, personagem, vegetação, etc)
type Elemento struct {
	simbolo  rune
	cor      Cor
	corFundo Cor
	tangivel bool // Indica se o elemento bloqueia passagem
}

// Jogo contém o estado atual do jogo
type Jogo struct {
	Mutex               sync.Mutex
	Mapa                [][]Elemento // grade 2D representando o mapa
	PosX, PosY          int          // posição atual do personagem
	UltimoVisitado      Elemento     // elemento que estava na posição do personagem antes de mover
	StatusMsg           string       // mensagem para a barra de status
	Inimigos            []*Inimigos  // Agora usamos um slice de ponteiros para Inimigo
	Vida                int
	UltimoDano          time.Time
	VegetacoesColetadas int
	TempoRestante       int
	VegChan             chan int // Canal para comunicação de vegetações coletadas
	GameOver            bool
}

type Inimigos struct {
	X, Y  int
	Ativo bool
}

// Elementos visuais do jogo
var (
	Personagem = Elemento{'☺', CorCinzaEscuro, CorPadrao, true}
	Inimigo    = Elemento{'☠', CorVermelho, CorPadrao, true}
	Parede     = Elemento{'▤', CorParede, CorFundoParede, true}
	Vegetacao  = Elemento{'♣', CorVerde, CorPadrao, false}
	Vazio      = Elemento{' ', CorPadrao, CorPadrao, false}
	Armadilha  = Elemento{'X', CorAmarelo, CorPadrao, true}
	Coracao    = Elemento{'♡', CorVermelho, CorPadrao, true}
)

// Cria e retorna uma nova instância do jogo
func jogoNovo() Jogo {
	// O ultimo elemento visitado é inicializado como vazio
	// pois o jogo começa com o personagem em uma posição vazia
	return Jogo{UltimoVisitado: Vazio, Vida: 3, UltimoDano: time.Now().Add(-10 * time.Second)}
}

// Lê um arquivo texto linha por linha e constrói o mapa do jogo
// Lê um arquivo texto linha por linha e constrói o mapa do jogo
func jogoCarregarMapa(nome string, jogo *Jogo) error {
	arq, err := os.Open(nome)
	if err != nil {
		return fmt.Errorf("erro ao abrir mapa: %v", err)
	}
	defer arq.Close()

	scanner := bufio.NewScanner(arq)
	jogo.Inimigos = []*Inimigos{} // Inicializa slice de inimigos
	y := 0

	for scanner.Scan() {
		linha := scanner.Text()
		var linhaElems []Elemento

		for x, ch := range linha {
			elem := Vazio // Valor padrão

			switch ch {
			case Parede.simbolo:
				elem = Parede
			case Inimigo.simbolo:
				// Adiciona novo inimigo à lista
				jogo.Inimigos = append(jogo.Inimigos, &Inimigos{X: x, Y: y, Ativo: true})
				elem = Vazio // Não coloca o inimigo no mapa inicial
			case Vegetacao.simbolo:
				elem = Vegetacao
			case Armadilha.simbolo:
				elem = Armadilha
			case Personagem.simbolo:
				jogo.PosX, jogo.PosY = x, y
				elem = Vazio // Personagem é desenhado separadamente
			}

			linhaElems = append(linhaElems, elem)
		}

		jogo.Mapa = append(jogo.Mapa, linhaElems)
		y++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("erro ao ler mapa: %v", err)
	}

	// Verificação básica
	if len(jogo.Mapa) == 0 {
		return fmt.Errorf("mapa vazio")
	}
	if jogo.PosX == 0 && jogo.PosY == 0 {
		return fmt.Errorf("personagem não encontrado no mapa")
	}

	return nil
}

// Verifica se o personagem pode se mover para a posição (x, y)
func jogoPodeMoverPara(jogo *Jogo, x, y int) bool {
	if y < 0 || y >= len(jogo.Mapa) || x < 0 || x >= len(jogo.Mapa[y]) {
		return false
	}
	return !jogo.Mapa[y][x].tangivel && jogo.Mapa[y][x].simbolo != Inimigo.simbolo
}

// Move um elemento para a nova posição
func jogoMoverElemento(jogo *Jogo, x, y, dx, dy int) {
	jogo.Mutex.Lock()
	defer jogo.Mutex.Unlock()

	nx, ny := x+dx, y+dy

	// Obtem elemento atual na posição
	elemento := jogo.Mapa[y][x] // guarda o conteúdo atual da posição

	jogo.Mapa[y][x] = jogo.UltimoVisitado   // restaura o conteúdo anterior
	jogo.UltimoVisitado = jogo.Mapa[ny][nx] // guarda o conteúdo atual da nova posição
	jogo.Mapa[ny][nx] = elemento            // move o elemento
}

// Move-se aleatoriamente pelo mapa e persegue o jogador se estiver próximo
func inimigoPatrulhar(jogo *Jogo, posicaoJogador chan [2]int, done chan struct{}, inimigo *Inimigos) {
	timeout := time.NewTicker(500 * time.Millisecond)
	defer timeout.Stop()

	for {
		select {
		case pos := <-posicaoJogador:
			if !inimigo.Ativo {
				continue
			}

			// Cálculo de direção (corrigido)
			dx := 0
			dy := 0

			if pos[0] < inimigo.X {
				dx = -1
			} else if pos[0] > inimigo.X {
				dx = 1
			}

			if pos[1] < inimigo.Y {
				dy = -1
			} else if pos[1] > inimigo.Y {
				dy = 1
			}

			jogo.Mutex.Lock()
			novaX, novaY := inimigo.X+dx, inimigo.Y+dy

			if novaX >= 0 && novaX < len(jogo.Mapa[0]) &&
				novaY >= 0 && novaY < len(jogo.Mapa) &&
				!jogo.Mapa[novaY][novaX].tangivel {

				// Atualiza mapa
				jogo.Mapa[inimigo.Y][inimigo.X] = Vazio
				inimigo.X, inimigo.Y = novaX, novaY
				jogo.Mapa[inimigo.Y][inimigo.X] = Inimigo
			}
			jogo.Mutex.Unlock()

		case <-timeout.C:
			if !inimigo.Ativo {
				continue
			}

			jogo.Mutex.Lock()
			// Movimento aleatório válido
			for tentativa := 0; tentativa < 5; tentativa++ {
				dx, dy := rand.Intn(3)-1, rand.Intn(3)-1 // -1, 0, ou 1
				novaX, novaY := inimigo.X+dx, inimigo.Y+dy

				if novaX >= 0 && novaX < len(jogo.Mapa[0]) &&
					novaY >= 0 && novaY < len(jogo.Mapa) &&
					!jogo.Mapa[novaY][novaX].tangivel {

					jogo.Mapa[inimigo.Y][inimigo.X] = Vazio
					inimigo.X, inimigo.Y = novaX, novaY
					jogo.Mapa[inimigo.Y][inimigo.X] = Inimigo
					break
				}
			}
			jogo.Mutex.Unlock()

		case <-done:
			jogo.Mutex.Lock()
			inimigo.Ativo = false
			jogo.Mapa[inimigo.Y][inimigo.X] = Vazio
			jogo.Mutex.Unlock()
			return
		}
	}
}

func estaProximo(posJogador [2]int, x, y int) bool {
	dx := posJogador[0] - x
	dy := posJogador[1] - y
	return dx*dx+dy*dy <= 25 // Distância <= 5 células (raio quadrado)
}

// Ativa-se quando o jogador passa próximo e desativa após 3 segundos.
// Usa um canal para notificar o jogador (ex: "Você caiu em uma armadilha!").
func armadilha(jogo *Jogo, ativar <-chan bool, done <-chan struct{}) {
	var armadilhaAtiva bool

	for {
		select {
		case ativacao := <-ativar:
			if !armadilhaAtiva && ativacao { // Só ativa se não estiver já ativa
				jogo.Mutex.Lock()
				// Encontra todas as armadilhas no mapa
				for y := range jogo.Mapa {
					for x := range jogo.Mapa[y] {
						if jogo.Mapa[y][x].simbolo == Armadilha.simbolo {
							jogo.Mapa[y][x].simbolo = '!'
							jogo.Mapa[y][x].cor = CorVermelho
						}
					}
				}
				armadilhaAtiva = true
				jogo.StatusMsg = "Armadilha ativada!"
				jogo.Mutex.Unlock()

				// Desativa após 3 segundos
				time.AfterFunc(3*time.Second, func() {
					jogo.Mutex.Lock()
					defer jogo.Mutex.Unlock()
					for y := range jogo.Mapa {
						for x := range jogo.Mapa[y] {
							if jogo.Mapa[y][x].simbolo == '!' {
								jogo.Mapa[y][x].simbolo = Armadilha.simbolo
								jogo.Mapa[y][x].cor = CorAmarelo
							}
						}
					}
					armadilhaAtiva = false
				})
			}

		case <-done:
			return
		}
	}
}

func jogadorPertoDeArmadilha(jogo *Jogo, x, y int) bool {
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			nx, ny := x+dx, y+dy
			if ny >= 0 && ny < len(jogo.Mapa) && nx >= 0 && nx < len(jogo.Mapa[ny]) {
				if jogo.Mapa[ny][nx].simbolo == Armadilha.simbolo {
					return true
				}
			}
		}
	}
	return false
}

func timerJogo(jogo *Jogo, done chan struct{}) {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			jogo.Mutex.Lock()
			jogo.TempoRestante--
			jogo.Mutex.Unlock()
		case <-timeout:
			jogo.Mutex.Lock()
			jogo.GameOver = true
			jogo.StatusMsg = fmt.Sprintf("Tempo esgotado! Vegetacoes coletadas: %d", jogo.VegetacoesColetadas)
			jogo.Mutex.Unlock()
			close(done)
			return
		case <-done:
			return
		}
	}
}
