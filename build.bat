@echo off
REM build.bat - Script de build para Windows

REM Inicializa o módulo go, se não existir
IF NOT EXIST go.mod (
    echo Inicializando go.mod...
    go mod init jogo
    go get -u github.com/nsf/termbox-go
)

REM Compila o projeto
echo Compilando...
go build -o jogo.exe

REM Fim
exit /b
