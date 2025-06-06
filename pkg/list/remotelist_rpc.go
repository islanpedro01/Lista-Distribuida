package remotelist

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Configurações de nomes de arquivo (você pode ajustar para outro diretório se quiser)
const (
	SnapshotFilename = "snapshot.dat"
	LogFilename      = "operations.log"
	// A cada quantos segundos vamos forçar a criação de um novo snapshot:
	SnapshotInterval = 30 * time.Second
)

type RemoteList struct {
	mu        sync.RWMutex
	lists     map[string][]int
	logFile   *os.File // arquivo de log aberto para append
	snapMutex sync.Mutex
	// snapMutex protege apenas a parte de criar snapshot (para duplicar o map em memória)
}

// func (l *RemoteList) Append(value int, reply *bool) error {
// 	l.mu.Lock()
// 	defer l.mu.Unlock()
// 	l.list = append(l.list, value)
// 	fmt.Println(l.list)
// 	l.size++
// 	*reply = true
// 	return nil
// }

// func (l *RemoteList) Remove(arg int, reply *int) error {
// 	l.mu.Lock()
// 	defer l.mu.Unlock()
// 	if len(l.list) > 0 {
// 		*reply = l.list[len(l.list)-1]
// 		l.list = l.list[:len(l.list)-1]
// 		fmt.Println(l.list)
// 	} else {
// 		return errors.New("empty list")
// 	}
// 	return nil
// }

func NewRemoteList() (*RemoteList, error) {
		rl := &RemoteList{
		lists: make(map[string][]int),
}

// 1) Carrega snapshot (se existir)
	if err := rl.loadSnapshot(); err != nil {
		return nil, fmt.Errorf("falha ao carregar snapshot: %w", err)
	}

	// 2) Abre (ou cria) o arquivo de log para leitura+escrita
	logFile, err := os.OpenFile(LogFilename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir/criar log: %w", err)
	}
	rl.logFile = logFile

	// 3) Aplica as operações que estão no log (desde o início)
	if err := rl.applyLog(); err != nil {
		return nil, fmt.Errorf("falha ao aplicar operações do log: %w", err)
	}

	// 4) Inicia a goroutine de snapshot periódico
	go rl.snapshotLoop()

	return rl, nil
}


// Carregar Snapshot do Disco
func (rl *RemoteList) loadSnapshot() error {
	if _, err := os.Stat(SnapshotFilename); os.IsNotExist(err) {
		// não há snapshot: nada a fazer
		return nil
	}
	f, err := os.Open(SnapshotFilename)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	var data map[string][]int
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("decode snapshot: %w", err)
	}
	rl.lists = data
	return nil
}

// ------------------------------
// Ler e aplicar log de operações (append/remove) que já estavam gravadas.
func (rl *RemoteList) applyLog() error {
	f, err := os.Open(LogFilename)
	if os.IsNotExist(err) {
		// não existe log (talvez primeira vez). Cria um vazio.
		return nil
	} else if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Esperamos linhas no formato "APPEND:<listID>:<valor>"
		// ou "REMOVE:<listID>"
		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue // linha inválida, ignora
		}
		op := parts[0]
		listID := parts[1]
		switch op {
		case "APPEND":
			if len(parts) != 3 {
				continue
			}
			// converter valor para int
			var v int
			fmt.Sscanf(parts[2], "%d", &v)
			rl.applyAppend(listID, v)
		case "REMOVE":
			// apenas listID
			rl.applyRemove(listID)
		default:
			// linha desconhecida: ignora
		}
	}
	return scanner.Err()
}

// applyRemove remove o último elemento de uma lista (sem verificar existência).
func (rl *RemoteList) applyRemove(listID string) {
	slice, ok := rl.lists[listID]
	if !ok || len(slice) == 0 {
		return
	}
	rl.lists[listID] = slice[:len(slice)-1]
}


// ------------------------------
// Funções auxiliares para gravar no log (sempre sob lock de escrita)

// appendToLog registra "APPEND:<listID>:<value>\n"
func (rl *RemoteList) appendToLog(listID string, value int) error {
	line := fmt.Sprintf("APPEND:%s:%d\n", listID, value)
	if _, err := rl.logFile.WriteString(line); err != nil {
		return err
	}
	return rl.logFile.Sync()
}

// removeToLog registra "REMOVE:<listID>\n"
func (rl *RemoteList) removeToLog(listID string) error {
	line := fmt.Sprintf("REMOVE:%s\n", listID)
	if _, err := rl.logFile.WriteString(line); err != nil {
		return err
	}
	return rl.logFile.Sync()
}

// ------------------------------
// Goroutine que dispara snapshots periodicamente
func (rl *RemoteList) snapshotLoop() {
	ticker := time.NewTicker(SnapshotInterval)
	for range ticker.C {
		if err := rl.takeSnapshot(); err != nil {
			fmt.Println("erro ao salvar snapshot:", err)
		}
	}
}

// takeSnapshot faz um “dump” completo de rl.lists em JSON, com cópia rápida sob RLock.
// Em seguida, trunca o arquivo de log original.
func (rl *RemoteList) takeSnapshot() error {
	// 1) Faz cópia rápida de rl.lists sob lock de leitura
	rl.mu.RLock()
	copia := make(map[string][]int, len(rl.lists))
	for k, slice := range rl.lists {
		// copiar o slice inteiro
		novaSlice := make([]int, len(slice))
		copy(novaSlice, slice)
		copia[k] = novaSlice
	}
	rl.mu.RUnlock()

	// 2) Serializar a cópia num arquivo temporário
	tempName := SnapshotFilename + ".tmp"
	f, err := os.OpenFile(tempName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("abrir snapshot tmp: %w", err)
	}
	encoder := json.NewEncoder(f)
	if err := encoder.Encode(copia); err != nil {
		f.Close()
		return fmt.Errorf("encode snapshot: %w", err)
	}
	f.Close()

	// 3) Renomear tmp para o arquivo definitivo (rename é atômico na maioria dos SO)
	if err := os.Rename(tempName, SnapshotFilename); err != nil {
		return fmt.Errorf("rename snapshot: %w", err)
	}

	// 4) Agora que temos snapshot atualizado, truncamos o log
	rl.snapMutex.Lock() // garante que apenas uma snapshot está truncando log
	defer rl.snapMutex.Unlock()

	// Fecha o log atual, renomeia e recria um novo vazio:
	oldLog := LogFilename
	backupLog := LogFilename + ".old"
	rl.logFile.Close()

	// Se já existir um backup anterior, remove (para não acumular .old.old, etc)
	os.Remove(backupLog)
	// Renomeia log.dat → log.dat.old
	if err := os.Rename(oldLog, backupLog); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: não consegui renomear log anterior: %v\n", err)
	}
	// Cria novo log vazio
	newLog, err := os.OpenFile(LogFilename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("criar novo log: %w", err)
	}
	rl.logFile = newLog
	return nil
}

// --------------------------------------------------------------------------------
// Métodos expostos via RPC. Note que cada assinatura é (args, reply *X) error.

// Append insere "value" ao final da lista identificada por args.ListID.
// Se a lista não existir, cria nova. Em seguida, grava no log.
// Lock: escrita.
func (rl *RemoteList) Append(args *AppendArgs, reply *AppendReply) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Atualiza em memória
	slice, ok := rl.lists[args.ListID]
	if !ok {
		slice = []int{}
	}
	slice = append(slice, args.Value)
	rl.lists[args.ListID] = slice

	// Grava no arquivo de log
	if err := rl.appendToLog(args.ListID, args.Value); err != nil {
		// Se falhar no log, poderíamos reverter em memória... mas deixamos como está e retornamos erro
		return fmt.Errorf("falha ao gravar no log: %w", err)
	}

	reply.Success = true
	return nil
}

// Get retorna o elemento na posição args.Index da lista args.ListID.
// Erro se a lista não existir ou index fora de intervalo.
// Lock: leitura.
func (rl *RemoteList) Get(args *GetArgs, reply *GetReply) error {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	slice, ok := rl.lists[args.ListID]
	if !ok {
		return errors.New("lista não existe")
	}
	if args.Index < 0 || args.Index >= len(slice) {
		return errors.New("index fora de intervalo")
	}

	reply.Value = slice[args.Index]
	return nil
}

// Remove retira e retorna o último elemento da lista args.ListID.
// Erro se lista não existir ou vazia.
// Lock: escrita.
func (rl *RemoteList) Remove(args *RemoveArgs, reply *RemoveReply) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	slice, ok := rl.lists[args.ListID]
	if !ok || len(slice) == 0 {
		return errors.New("lista vazia ou não existe")
	}
	last := slice[len(slice)-1]
	rl.lists[args.ListID] = slice[:len(slice)-1]

	// Grava no log
	if err := rl.removeToLog(args.ListID); err != nil {
		return fmt.Errorf("falha ao gravar remove no log: %w", err)
	}

	reply.Value = last
	return nil
}

// Size retorna o número de elementos na lista args.ListID.
// Se a lista não existir, devolve 0 (ou poderíamos criar a lista vazia?)
// Lock: leitura.
func (rl *RemoteList) Size(args *SizeArgs, reply *SizeReply) error {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	slice, ok := rl.lists[args.ListID]
	if !ok {
		reply.Size = 0
		return nil
	}
	reply.Size = len(slice)
	return nil
}