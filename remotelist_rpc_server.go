package main

import (
	"fmt"
	"net"
	"net/rpc"
	"pkg/remotelist"
	"os"
	"os/signal"
	"syscall"
)

// func main() {
// 	list := new(remotelist.RemoteList)
// 	rpcs := rpc.NewServer()
// 	rpcs.Register(list)
// 	l, e := net.Listen("tcp", "[localhost]:5000")
// 	defer l.Close()
// 	if e != nil {
// 		fmt.Println("listen error:", e)
// 	}
// 	for {
// 		conn, err := l.Accept()
// 		if err == nil {
// 			go rpcs.ServeConn(conn)
// 		} else {
// 			break
// 		}
// 	}
// }

func main(){

		// 1) Instanciamos o RemoteList (ele já carrega snapshot + log + inicia o snapshotLoop)
	rl, err := remotelist.NewRemoteList()
	if err != nil {
		fmt.Println("Erro ao inicializar RemoteList:", err)
		return
	}

		// 2) Registramos no servidor RPC
	rpcServer := rpc.NewServer()
	if err := rpcServer.Register(rl); err != nil {
		fmt.Println("Erro ao registrar RemoteList no RPC:", err)
		return
	}

		// 3) Começamos a escutar conexões TCP na porta 5000
	listener, err := net.Listen("tcp", ":5000")
	if err != nil {
		fmt.Println("Erro ao abrir listener TCP:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Servidor RPC rodando em :5000")

	// 4) Captura SIGINT/SIGTERM para encerrar graciosamente (se quiser)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 5) Aceita conexões em background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Erro ao aceitar conexão:", err)
				continue
			}
			// Cada conexão fica em sua goroutine
			go rpcServer.ServeConn(conn)
		}
	}()

		// 6) Espera sinal de interrupção para encerrar
	<-sigCh
	fmt.Println("Servidor interrompido. Fechando arquivos de log...")
	rl.Cleanup() // método opcional para fechar logFile, se quiser
}
