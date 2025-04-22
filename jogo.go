// jogo.go - Funções para manipular os elementos do jogo, como carregar o mapa e mover o personagem
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
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
	Mutex          sync.Mutex
	Mapa           [][]Elemento // grade 2D representando o mapa
	PosX, PosY     int          // posição atual do personagem
	UltimoVisitado Elemento     // elemento que estava na posição do personagem antes de mover
	StatusMsg      string       // mensagem para a barra de status
	Inimigos       []*Inimigos  // Agora usamos um slice de ponteiros para Inimigo
	Vidas          int
	GameOver       bool
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
	ItemCura   = Elemento{'♥', CorAzul, CorPadrao, false}
)

// Cria e retorna uma nova instância do jogo
func jogoNovo() Jogo {
	// O ultimo elemento visitado é inicializado como vazio
	// pois o jogo começa com o personagem em uma posição vazia
	return Jogo{
		UltimoVisitado: Vazio,
		Vidas:          3,
		GameOver:       false,
	}
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
			elem := Vazio
			switch ch {
			case Parede.simbolo:
				elem = Parede
			case Inimigo.simbolo:
				jogo.Inimigos = append(jogo.Inimigos, &Inimigos{X: x, Y: y, Ativo: true})
				elem = Vazio
			case Vegetacao.simbolo:
				elem = Vegetacao
			case Armadilha.simbolo:
				elem = Armadilha
			case Personagem.simbolo:
				jogo.PosX, jogo.PosY = x, y
				elem = Vazio
			case ItemCura.simbolo:
				elem = ItemCura
			}
			linhaElems = append(linhaElems, elem)
		}
		jogo.Mapa = append(jogo.Mapa, linhaElems)
		y++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("erro ao ler mapa: %v", err)
	}

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

	// Preserva vegetação ao mover
	elementoOriginal := jogo.Mapa[ny][nx]
	if elementoOriginal.simbolo == Vegetacao.simbolo {
		jogo.UltimoVisitado = Vegetacao
	} else {
		jogo.UltimoVisitado = Vazio
	}

	// Move o elemento
	jogo.Mapa[y][x] = jogo.UltimoVisitado
	jogo.UltimoVisitado = jogo.Mapa[ny][nx]
	jogo.Mapa[ny][nx] = Inimigo // Mantém o símbolo do inimigo
}

// Move-se aleatoriamente pelo mapa e persegue o jogador se estiver próximo
// Move-se aleatoriamente pelo mapa e persegue o jogador se estiver próximo
func inimigoPatrulhar(jogo *Jogo, posicaoJogador chan [2]int, done chan struct{}, inimigo *Inimigos) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			jogo.Mutex.Lock()
			inimigo.Ativo = false
			jogo.Mutex.Unlock()
			return

		case <-ticker.C:
			if !inimigo.Ativo {
				continue
			}
			// Movimento aleatório
			jogo.Mutex.Lock()
			for tentativa := 0; tentativa < 5; tentativa++ {
				dx, dy := rand.Intn(3)-1, rand.Intn(3)-1
				novaX, novaY := inimigo.X+dx, inimigo.Y+dy
				if novaX >= 0 && novaX < len(jogo.Mapa[0]) &&
					novaY >= 0 && novaY < len(jogo.Mapa) &&
					jogoPodeMoverPara(jogo, novaX, novaY) {
					inimigo.X, inimigo.Y = novaX, novaY
					break
				}
			}
			jogo.Mutex.Unlock()

			// Verifica posição do jogador (não bloqueante)
			select {
			case pos, ok := <-posicaoJogador:
				if !ok || !inimigo.Ativo {
					continue
				}
				if estaProximo(pos, inimigo.X, inimigo.Y) {
					jogo.Mutex.Lock()
					dx, dy := 0, 0
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
					novaX, novaY := inimigo.X+dx, inimigo.Y+dy
					if novaX >= 0 && novaX < len(jogo.Mapa[0]) &&
						novaY >= 0 && novaY < len(jogo.Mapa) &&
						jogoPodeMoverPara(jogo, novaX, novaY) {
						inimigo.X, inimigo.Y = novaX, novaY
					}
					jogo.Mutex.Unlock()
				}
			default:
			}
		}
	}
}

func estaProximo(posJogador [2]int, x, y int) bool {
	dx := posJogador[0] - x
	dy := posJogador[1] - y
	return dx*dx+dy*dy <= 25 // Distância <= 5 células (raio quadrado)
}

// Aparece e desaparece em intervalos aleatórios.
func itemCura(jogo *Jogo, done <-chan struct{}) {
	for {
		select {
		case <-time.After(time.Duration(rand.Intn(15)+10) * time.Second): // 10-25 segundos
			jogo.Mutex.Lock()

			// Lista de posições válidas (vazias e não próximas ao jogador)
			var posicoesValidas [][2]int
			for y := range jogo.Mapa {
				for x := range jogo.Mapa[y] {
					if jogo.Mapa[y][x].simbolo == Vazio.simbolo {
						// Verifica distância do jogador
						dx := x - jogo.PosX
						dy := y - jogo.PosY
						if dx*dx+dy*dy > 16 { // Pelo menos 4 células de distância
							posicoesValidas = append(posicoesValidas, [2]int{x, y})
						}
					}
				}
			}

			if len(posicoesValidas) > 0 {
				pos := posicoesValidas[rand.Intn(len(posicoesValidas))]
				jogo.Mapa[pos[1]][pos[0]] = ItemCura
				jogo.StatusMsg = "Item de cura apareceu em (" + strconv.Itoa(pos[0]) + "," + strconv.Itoa(pos[1]) + ")"

				// Desaparece após 5 segundos
				time.AfterFunc(5*time.Second, func() {
					jogo.Mutex.Lock()
					if jogo.Mapa[pos[1]][pos[0]].simbolo == ItemCura.simbolo {
						jogo.Mapa[pos[1]][pos[0]] = Vazio
					}
					jogo.Mutex.Unlock()
				})
			}

			jogo.Mutex.Unlock()

		case <-done:
			return
		}
	}
}

// Ativa-se quando o jogador passa próximo e desativa após 3 segundos.
// Usa um canal para notificar o jogador (ex: "Você caiu em uma armadilha!").
func armadilha(jogo *Jogo, ativar <-chan bool, done <-chan struct{}) {
	var armadilhaAtiva bool

	for {
		select {
		case ativacao, ok := <-ativar:
			if !ok {
				return
			}
			if !armadilhaAtiva && ativacao {
				jogo.Mutex.Lock()
				for y := range jogo.Mapa {
					for x := range jogo.Mapa[y] {
						if jogo.Mapa[y][x].simbolo == Armadilha.simbolo {
							jogo.Mutex[y][x].simbolo = '!'
							jogo.Mapa[y][x].cor = CorVermelho
						}
					}
				}
				armadilhaAtiva = true
				jogo.StatusMsg = "Armadilha ativada!"
				jogo.Mutex.Unlock()

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
					jogo.StatusMsg = ""
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
