package main

import (
	"LISTA-DISTRIBUIDA/pkg/remotelist"
	"fmt"
	"log"
	"math/rand"
	"net/rpc"
	"sync"
	"time"
)

const (
	numClients = 3
	numOps     = 5
)

func runClient(id int, wg *sync.WaitGroup) {
	defer wg.Done()

	// Cada cliente usa sua própria conexão
	client, err := rpc.Dial("tcp", ":5000")
	if err != nil {
		log.Printf("[Client %d] Erro ao conectar: %v", id, err)
		return
	}
	defer client.Close()

	// listID := fmt.Sprintf("lista%d", id)
	listID := "concorrente"
	logPrefix := fmt.Sprintf("[Client %d]", id)

	// Realiza várias operações aleatórias
	for i := 0; i < numOps; i++ {
		op := rand.Intn(4)
		switch op {
		case 0: // Append
			val := rand.Intn(1000)
			args := &remotelist.AppendArgs{ListID: listID, Value: val}
			var rep remotelist.AppendReply
			if err := client.Call("RemoteList.Append", args, &rep); err != nil {
				log.Printf("%s Erro em Append: %v", logPrefix, err)
				} else {
			log.Println(logPrefix, "Append:", val, "Na lista:", listID)
		}

		case 1: // Remove
			args := &remotelist.RemoveArgs{ListID: listID}
			var rep remotelist.RemoveReply
			err := client.Call("RemoteList.Remove", args, &rep)
			if err != nil {
				log.Printf("%s Remove (lista possivelmente vazia): %v", logPrefix, err)
			}else {
			log.Println( logPrefix, "Remove:", rep.Value,"Na lista:", listID)
		}
		case 2: // Size
			args := &remotelist.SizeArgs{ListID: listID}
			var rep remotelist.SizeReply
			if err := client.Call("RemoteList.Size", args, &rep); err == nil {
				log.Printf("%s Size = %d", logPrefix, rep.Size)
			}
		case 3: // Get
			// Primeiro vê o tamanho
			sizeArgs := &remotelist.SizeArgs{ListID: listID}
			var sizeRep remotelist.SizeReply
			if err := client.Call("RemoteList.Size", sizeArgs, &sizeRep); err != nil || sizeRep.Size == 0 {
				continue
			}
			index := rand.Intn(sizeRep.Size)
			args := &remotelist.GetArgs{ListID: listID, Index: index}
			var rep remotelist.GetReply
			if err := client.Call("RemoteList.Get", args, &rep); err == nil {
				log.Printf("%s Get[%d] = %d", logPrefix, index, rep.Value)
			}
		}
		time.Sleep(time.Millisecond * time.Duration(50+rand.Intn(100))) // espera entre ops
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	var wg sync.WaitGroup

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go runClient(i, &wg)
	}

	wg.Wait()
	fmt.Println("Todos os clientes terminaram.")
}
