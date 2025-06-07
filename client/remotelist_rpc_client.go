package main

import (
	"fmt"
	"log"
	"net/rpc"
	"LISTA-DISTRIBUIDA/pkg/remotelist"
)

func main() {
	// 1) Cria conexão RPC com o servidor em localhost:5000
	client, err := rpc.Dial("tcp", ":5000")
	if err != nil {
		fmt.Print("dialing:", err)
	}

// 2) Exemplo de uso: vamos criar duas listas: "listaA" e "listaB"
	// e fazer algumas operações em cada uma.

	// Append em "listaA"
	appendArgs := &remotelist.AppendArgs{
		ListID: "listaA",
		Value:  10,
	}
	var appendRep remotelist.AppendReply
	if err := client.Call("RemoteList.Append", appendArgs, &appendRep); err != nil {
		log.Println("Erro em Append:", err)
	} else {
		fmt.Println("Append em listaA:", appendRep.Success)
	}

	// Append em "listaA" de novo
	appendArgs = &remotelist.AppendArgs{
		ListID: "listaA",
		Value:  20,
	}
	if err := client.Call("RemoteList.Append", appendArgs, &appendRep); err != nil {
		log.Println("Erro em Append:", err)
	} else {
		fmt.Println("Append 20 em listaA:", appendRep.Success)
	}

	// Append em "listaB"
	appendArgs = &remotelist.AppendArgs{
		ListID: "listaB",
		Value:  100,
	}
	if err := client.Call("RemoteList.Append", appendArgs, &appendRep); err != nil {
		log.Println("Erro em Append:", err)
	} else {
		fmt.Println("Append 100 em listaB:", appendRep.Success)
	}

	// Get( listaA, índice 1 ) -> deve retornar 20
	getArgs := &remotelist.GetArgs{
		ListID: "listaA",
		Index:  1,
	}
	var getRep remotelist.GetReply
	if err := client.Call("RemoteList.Get", getArgs, &getRep); err != nil {
		log.Println("Erro em Get:", err)
	} else {
		fmt.Printf("Get(listaA, 1) = %d\n", getRep.Value)
	}

	// Size(listaA) -> deve retornar 2
	sizeArgs := &remotelist.SizeArgs{ListID: "listaA"}
	var sizeRep remotelist.SizeReply
	if err := client.Call("RemoteList.Size", sizeArgs, &sizeRep); err != nil {
		log.Println("Erro em Size:", err)
	} else {
		fmt.Printf("Size(listaA) = %d\n", sizeRep.Size)
	}

	// Remove(listaA) -> deve retornar 20
	removeArgs := &remotelist.RemoveArgs{ListID: "listaA"}
	var removeRep remotelist.RemoveReply
	if err := client.Call("RemoteList.Remove", removeArgs, &removeRep); err != nil {
		log.Println("Erro em Remove:", err)
	} else {
		fmt.Printf("Remove(listaA) = %d\n", removeRep.Value)
	}

	// Size(listaA) de novo → deve retornar 1
	if err := client.Call("RemoteList.Size", sizeArgs, &sizeRep); err != nil {
		log.Println("Erro em Size:", err)
	} else {
		fmt.Printf("Size(listaA) depois de Remove = %d\n", sizeRep.Size)
	}

	// Remove(listaA) outra vez → retorna 10
	if err := client.Call("RemoteList.Remove", removeArgs, &removeRep); err != nil {
		log.Println("Erro em Remove:", err)
	} else {
		fmt.Printf("Remove(listaA) = %d\n", removeRep.Value)
	}

	// Tentamos Remove(listaA) de lista vazia → deve dar erro
	if err := client.Call("RemoteList.Remove", removeArgs, &removeRep); err != nil {
		fmt.Println("Remove em lista vazia gerou erro (esperado):", err)
	} else {
		fmt.Printf("Remove(listaA) = %d\n", removeRep.Value)
	}
}

