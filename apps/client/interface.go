package main

import (
	"fppd-jogo/logica_jogo"
	"fmt"
	"sort"
	"github.com/nsf/termbox-go"
)

type Jogo = logica_jogo.Jogo
type EventoTeclado = logica_jogo.EventoTeclado
type Cor = termbox.Attribute

const (
	CorPadrao   Cor = termbox.ColorDefault
	CorAmarelo  Cor = termbox.ColorYellow
	CorTexto    Cor = termbox.ColorDarkGray
)

func Iniciar() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
}

func Finalizar() {
	termbox.Close()
}

func LerEventoTeclado() EventoTeclado {
	ev := termbox.PollEvent()
	if ev.Type != termbox.EventKey {
		return EventoTeclado{}
	}
	if ev.Key == termbox.KeyEsc {
		return EventoTeclado{Tipo: "sair"}
	}
	if ev.Ch == 'e' || ev.Ch == 'E' {
		return EventoTeclado{Tipo: "interagir"}
	}
	if ev.Ch == 'w' || ev.Ch == 'a' || ev.Ch == 's' || ev.Ch == 'd' {
		return EventoTeclado{Tipo: "mover", Tecla: ev.Ch}
	}
	return EventoTeclado{}
}

func DesenharJogo(jogo *Jogo) {
	termbox.Clear(CorPadrao, CorPadrao)

	if jogo == nil {
		termbox.Flush()
		return
	}

	// 1. Desenha o mapa base
	for y, linha := range jogo.Mapa {
		for x, elem := range linha {
			termbox.SetCell(x, y, elem.Simbolo, elem.Cor, elem.CorFundo)
		}
	}

	// 2. Desenha todos os jogadores por cima do mapa
	for name, pState := range jogo.Players {
		var char logica_jogo.Elemento
		if name == jogo.MyName {
			char = logica_jogo.Personagem
		} else {
			char = logica_jogo.OutroJogador
		}
		termbox.SetCell(pState.X, pState.Y, char.Simbolo, char.Cor, char.CorFundo)
	}

	desenharBarraDeStatus(jogo)
	termbox.Flush()
}

func desenharBarraDeStatus(jogo *Jogo) {
	meuEstado, ok := jogo.Players[jogo.MyName]
	if !ok {
		return
	}

	linhaYBase := len(jogo.Mapa) + 1

	status := fmt.Sprintf("%s | Sua pontuação: %d", jogo.StatusMsg, meuEstado.VegetacoesColetadas)
	drawString(0, linhaYBase, status, CorTexto, CorPadrao)

	drawString(0, linhaYBase+1, "--- PONTUAÇÕES ---", CorAmarelo, CorPadrao)

	playerNames := make([]string, 0, len(jogo.Players))
	for name := range jogo.Players {
		playerNames = append(playerNames, name)
	}
	sort.Strings(playerNames)

	linhaPlacarY := linhaYBase + 2
	for _, name := range playerNames {
		pState := jogo.Players[name] 
		placarJogador := fmt.Sprintf("%s: %d", pState.Name, pState.VegetacoesColetadas)
		drawString(0, linhaPlacarY, placarJogador, CorTexto, CorPadrao)
		linhaPlacarY++ 
	}

	msg := "Use WASD para mover e E para interagir. ESC para sair."
	drawString(0, linhaPlacarY+1, msg, CorTexto, CorPadrao)
}

func drawString(x, y int, text string, fg, bg Cor) {
	for i, c := range text {
		termbox.SetCell(x+i, y, c, fg, bg)
	}
}
