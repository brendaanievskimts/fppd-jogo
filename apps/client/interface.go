package main
import (
	"fppd-jogo/logica_jogo"
	"fmt"
	"github.com/nsf/termbox-go"
)

type Jogo = logica_jogo.Jogo
type EventoTeclado = logica_jogo.EventoTeclado
type Cor = termbox.Attribute

const (
	CorPadrao   Cor = termbox.ColorDefault
	CorVermelho Cor = termbox.ColorRed
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

	for y, linha := range jogo.Mapa {
		for x, elem := range linha {
			termbox.SetCell(x, y, elem.Simbolo, elem.Cor, elem.CorFundo)
		}
	}

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

	status := fmt.Sprintf("%s | Mato: %d", jogo.StatusMsg, meuEstado.VegetacoesColetadas)
	for i, c := range status {
		termbox.SetCell(i, len(jogo.Mapa)+1, c, CorTexto, CorPadrao)
	}

	// Desenhar vida
	for i := 0; i < 3; i++ {
		coracao := '♥'
		cor := CorVermelho
		if i >= meuEstado.Vida {
			coracao = '♡'
			cor = CorTexto
		}
		termbox.SetCell(70+i*2, len(jogo.Mapa)+1, coracao, cor, CorPadrao)
	}

	msg := "Use WASD para mover e E para interagir. ESC para sair."
	for i, c := range msg {
		termbox.SetCell(i, len(jogo.Mapa)+3, c, CorTexto, CorPadrao)
	}
}
