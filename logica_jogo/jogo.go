package logica_jogo

import (
	"fppd-jogo/game"
	"fmt"
	"github.com/nsf/termbox-go"
)

type Cor = termbox.Attribute

type EventoTeclado struct {
	Tipo  string
	Tecla rune
}

type Elemento struct {
	Simbolo  rune
	Cor      Cor
	CorFundo Cor
	Tangivel bool
}

type Jogo struct {
	MyName    string
	Mapa      [][]Elemento
	Players   map[string]game.PlayerState
	StatusMsg string
}

var (
	// ALTERAÇÃO: Símbolos trocados por caracteres ASCII de largura padrão para evitar distorção.
	Personagem   = Elemento{'☺', termbox.ColorWhite, termbox.ColorDefault, true}
	OutroJogador = Elemento{'o', termbox.ColorYellow, termbox.ColorDefault, true}
	Parede       = Elemento{'▤', termbox.ColorDarkGray, termbox.ColorBlack, true}
	Vegetacao    = Elemento{'♣', termbox.ColorGreen, termbox.ColorDefault, false}
	Vazio        = Elemento{' ', termbox.ColorDefault, termbox.ColorDefault, false}
)

func NovoJogo(myName string) *Jogo {
	return &Jogo{
		MyName:  myName,
		Players: make(map[string]game.PlayerState),
	}
}

func (jogo *Jogo) ExecutarAcao(ev EventoTeclado) bool {
	switch ev.Tipo {
	case "sair":
		return false
	case "interagir":
		meuEstado, ok := jogo.Players[jogo.MyName]
		if ok {
			jogo.StatusMsg = fmt.Sprintf("Interagindo em (%d, %d)", meuEstado.X, meuEstado.Y)
		}
	case "mover":
		jogo.moverPersonagem(ev.Tecla)
	}
	return true
}

func (jogo *Jogo) moverPersonagem(tecla rune) {
	meuEstado, ok := jogo.Players[jogo.MyName]
	if !ok {
		return
	}

	dx, dy := 0, 0
	switch tecla {
	case 'w':
		dy = -1
	case 'a':
		dx = -1
	case 's':
		dy = 1
	case 'd':
		dx = 1
	}

	nx, ny := meuEstado.X+dx, meuEstado.Y+dy

	if jogo.podeMoverPara(nx, ny) {
		// Verifica se a nova posição contém uma vegetação
		if jogo.Mapa[ny][nx].Simbolo == Vegetacao.Simbolo {
			// ALTERAÇÃO: A linha que modifica o mapa localmente foi removida.
			// O cliente agora apenas atualiza a pontuação e informa o servidor.
			// A remoção visual da vegetação virá do estado autoritativo do servidor,
			// evitando o efeito de "piscar".
			meuEstado.VegetacoesColetadas++
		}
		// Atualiza a posição do jogador localmente para feedback imediato
		meuEstado.X = nx
		meuEstado.Y = ny
		jogo.Players[jogo.MyName] = meuEstado
	}
}

func (jogo *Jogo) podeMoverPara(x, y int) bool {
	// Verifica limites do mapa
	if y < 0 || y >= len(jogo.Mapa) || x < 0 || x >= len(jogo.Mapa[y]) {
		return false
	}
	// Verifica se o tile é uma parede
	if jogo.Mapa[y][x].Tangivel {
		return false
	}
	// Verifica colisão com outros jogadores
	for name, p := range jogo.Players {
		if name != jogo.MyName && p.X == x && p.Y == y {
			return false
		}
	}
	return true
}
