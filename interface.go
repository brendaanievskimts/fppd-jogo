// interface.go - Interface gráfica do jogo usando termbox
// O código abaixo implementa a interface gráfica do jogo usando a biblioteca termbox-go.
// A biblioteca termbox-go é uma biblioteca de interface de terminal que permite desenhar
// elementos na tela, capturar eventos do teclado e gerenciar a aparência do terminal.

package main

import (
	"fmt"

	"github.com/nsf/termbox-go"
)

// Define um tipo Cor para encapsuladar as cores do termbox
type Cor = termbox.Attribute

// Definições de cores utilizadas no jogo
const (
	CorPadrao   Cor = termbox.ColorDefault
	CorPreto        = termbox.ColorBlack
	CorVermelho     = termbox.ColorRed
	CorVerde        = termbox.ColorGreen
	CorAmarelo      = termbox.ColorYellow
	CorAzul         = termbox.ColorBlue
	CorMagenta      = termbox.ColorMagenta
	CorCiano        = termbox.ColorCyan
	CorBranco       = termbox.ColorWhite

	// Personalizações
	CorCinzaEscuro = termbox.ColorDarkGray
	CorTexto       = termbox.ColorDarkGray
	CorParede      = termbox.ColorBlack | termbox.AttrBold | termbox.AttrDim
	CorFundoParede = termbox.ColorDarkGray

	// Atributos de estilo (podem ser usados junto das cores)
	AttrNegrito    = termbox.AttrBold
	AttrSublinhado = termbox.AttrUnderline
	AttrPiscando   = termbox.AttrBlink // (se suportado pelo terminal)
	AttrInverso    = termbox.AttrReverse
)

// EventoTeclado representa uma ação detectada do teclado (como mover, sair ou interagir)
type EventoTeclado struct {
	Tipo  string // "sair", "interagir", "mover"
	Tecla rune   // Tecla pressionada, usada no caso de movimento
}

// Inicializa a interface gráfica usando termbox
func interfaceIniciar() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
}

// Encerra o uso da interface termbox
func interfaceFinalizar() {
	termbox.Close()
}

// Lê um evento do teclado e o traduz para um EventoTeclado
func interfaceLerEventoTeclado() EventoTeclado {
	ev := termbox.PollEvent()
	if ev.Type != termbox.EventKey {
		return EventoTeclado{}
	}
	// Ignora Caps Lock/Shift/outras teclas modificadoras
	if ev.Key == termbox.KeyEsc {
		return EventoTeclado{Tipo: "sair"}
	}
	if ev.Ch == 'e' || ev.Ch == 'E' {
		return EventoTeclado{Tipo: "interagir"}
	}
	// Aceita apenas WASD (minúsculas)
	if ev.Ch == 'w' || ev.Ch == 'a' || ev.Ch == 's' || ev.Ch == 'd' {
		return EventoTeclado{Tipo: "mover", Tecla: ev.Ch}
	}
	return EventoTeclado{} // Ignora outras teclas
}

// Renderiza todo o estado atual do jogo na tela
func interfaceDesenharJogo(jogo *Jogo) {
	interfaceLimparTela()
	// Desenha todos os elementos do mapa
	for y, linha := range jogo.Mapa {
		for x, elem := range linha {
			interfaceDesenharElemento(x, y, elem)
		}
	}

	// Desenha o personagem sobre o mapa
	interfaceDesenharElemento(jogo.PosX, jogo.PosY, Personagem)

	// Desenha a barra de status
	interfaceDesenharBarraDeStatus(jogo)

	// Força a atualização do terminal
	interfaceAtualizarTela()
}

// Limpa a tela do terminal
func interfaceLimparTela() {
	termbox.Clear(CorPadrao, CorPadrao)
}

// Força a atualização da tela do terminal com os dados desenhados
func interfaceAtualizarTela() {
	termbox.Flush()
}

// Desenha um elemento na posição (x, y)
func interfaceDesenharElemento(x, y int, elem Elemento) {
	termbox.SetCell(x, y, elem.simbolo, elem.cor, elem.corFundo)
}

// Exibe uma barra de status com informações úteis ao jogador
func interfaceDesenharBarraDeStatus(jogo *Jogo) {
	// Linha de status dinâmica

	status := fmt.Sprintf("%s | Mato: %d | Tempo: %ds",
		jogo.StatusMsg, jogo.VegetacoesColetadas, jogo.TempoRestante)

	for i, c := range status {
		termbox.SetCell(i, len(jogo.Mapa)+1, c, CorTexto, CorPadrao)
	}

	// Desenhar corações da vida (♥♥♥)
	for i := 0; i < 3; i++ {
		coracao := '♥'
		cor := CorVermelho
		if i >= jogo.Vida {
			coracao = '♡' // coração vazio
			cor = CorTexto
		}
		termbox.SetCell(70+i*2, len(jogo.Mapa)+1, coracao, cor, CorPadrao)
	}

	// Instruções fixas
	msg := "Use WASD para mover e E para interagir. ESC para sair."
	for i, c := range msg {
		termbox.SetCell(i, len(jogo.Mapa)+3, c, CorTexto, CorPadrao)
	}
}
