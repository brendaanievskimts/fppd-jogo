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
type Inimigos struct {
	X, Y  int
	Ativo bool
}
type AlienMovel struct {
	X, Y     int
	Subindo  bool
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
	Aliens				[]AlienMovel
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
	Alien      = Elemento{'Ψ', CorCiano, CorPadrao, true}
	ArmadilhaAtivada = Elemento{'█', CorVermelho, CorPadrao, true}
	ArmadilhaAlerta  = Elemento{'!', CorVermelho | AttrNegrito, CorPreto, true}
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
			case Alien.simbolo:
				jogo.Aliens = append(jogo.Aliens, AlienMovel{
					X: x, Y: y, Subindo: true,
				})
				elem = Vazio	
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
    posEntrada := [2]int{58, 22} // Posição fixa da entrada
    var alertaAtivo bool
    
    for {
        select {
        case ativacao := <-ativar:
            jogo.Mutex.Lock()
            
            if ativacao && !alertaAtivo {
                // Fase 1: Alerta (muda X para !)
                for y := range jogo.Mapa {
                    for x := range jogo.Mapa[y] {
                        if jogo.Mapa[y][x].simbolo == Armadilha.simbolo {
                            jogo.Mapa[y][x] = ArmadilhaAlerta
                        }
                    }
                }
                alertaAtivo = true
                jogo.StatusMsg = "ALERTA! Armadilha detectada nas proximidades!"
                
                // Fase 2: Fechamento permanente após 2 segundos
                time.AfterFunc(2*time.Second, func() {
                    jogo.Mutex.Lock()
                    defer jogo.Mutex.Unlock()
                    
                    // Fecha a entrada permanentemente
                    if jogo.Mapa[posEntrada[1]][posEntrada[0]].simbolo != ArmadilhaAtivada.simbolo {
                        jogo.Mapa[posEntrada[1]][posEntrada[0]] = ArmadilhaAtivada
                        jogo.StatusMsg = "BARULHO! A entrada (58,22) foi selada permanentemente!"
                    }
                })
            }
            
            jogo.Mutex.Unlock()

        case <-done:
            return
        }
    }
}

func jogadorPertoDeArmadilha(jogo *Jogo, x, y int) bool {
    // Verifica em um retângulo ao redor da posição (58,22)
    return x >= 55 && x <= 61 && y >= 19 && y <= 25
}

func timerJogo(jogo *Jogo, done chan struct{}) {
	timeout := time.After(60 * time.Second)
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
			return
		}
	}
}

func moverAlien(alien *AlienMovel, jogo *Jogo) {
	jogo.Mutex.Lock()
	defer jogo.Mutex.Unlock()

	dy := 1
	if !alien.Subindo {
		dy = -1
	}
	nx := alien.X
	ny := alien.Y + dy

	// Verifica se está dentro dos limites do mapa
	if ny < 0 || ny >= len(jogo.Mapa) || nx < 0 || nx >= len(jogo.Mapa[ny]) {
		alien.Subindo = !alien.Subindo
		return
	}

	// Colisão com o jogador
	if nx == jogo.PosX && ny == jogo.PosY {
		if time.Since(jogo.UltimoDano) > time.Second {
			jogo.Vida--
			jogo.UltimoDano = time.Now()
			jogo.StatusMsg = fmt.Sprintf("Ψ Alien te atingiu! Vida: %d", jogo.Vida)
			if jogo.Vida <= 0 {
				jogo.GameOver = true
				jogo.StatusMsg = "Você foi derrotado pelo Alien!"
			}
		}
		return
	}
	

	// Impede movimento para paredes
	if jogo.Mapa[ny][nx].tangivel {
		alien.Subindo = !alien.Subindo
		return
	}

	// Move o alien
	jogo.Mapa[alien.Y][alien.X] = Vazio
	jogo.Mapa[ny][nx] = Alien
	alien.X = nx
	alien.Y = ny
}




