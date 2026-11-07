package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// structs
type Tarefa struct {
	Titulo string `json:"titulo"`
	ID     int    `json:"id"`
	Feita  bool   `json:"feita"`
}

// Maps
var (
	tarefas   = map[int]Tarefa{}
	proximoID = 1
	mu        sync.Mutex
)

// Calcula a soma de dois números passados com o  query params da URL
func calcula(c *echo.Context) error {
	a, err := strconv.Atoi(c.QueryParam("a"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "a precisa ser um número"})
	}
	b, err := strconv.Atoi(c.QueryParam("b"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "b precisa ser um número"})
	}
	return c.JSON(http.StatusOK, map[string]int{"resultado": a + b})
}

// da olá, Mundo! com um id opcional
func ola(c *echo.Context) error {
	id := c.QueryParam("id")

	if id != "" {
		return c.JSON(http.StatusOK, map[string]string{"message": "Olá, Mundo!", "id": id})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "Olá, Mundo!"})
}

// lista todas as tarefas cadastradas, ordenadas pelo ID
func listarTarefas(c *echo.Context) error {
	filtro := c.QueryParam("feita")
	lista := []Tarefa{}

	mu.Lock()
	defer mu.Unlock()

	for _, t := range tarefas {
		if filtro == "true" && !t.Feita {
			continue
		}
		lista = append(lista, t)
	}

	// o map não guarda ordem, então ordenamos pelo ID
	sort.Slice(lista, func(i, j int) bool { return lista[i].ID < lista[j].ID })

	return c.JSON(http.StatusOK, lista)
}

// cria tarefas
func criarTarefa(c *echo.Context) error {
	var tarefa Tarefa

	if err := c.Bind(&tarefa); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "JSON inválido"})
	}

	if tarefa.Titulo == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "o título é obrigatório"})
	}

	mu.Lock()
	defer mu.Unlock()

	tarefa.ID = proximoID // o ID é gerado por nós, não vem de fora
	proximoID++
	tarefas[tarefa.ID] = tarefa

	if err := salvar(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "falha ao salvar tarefas"})
	}

	return c.JSON(http.StatusCreated, tarefa)
}

// busca uma tarefa pelo ID
func buscarTarefa(c *echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id precisa ser um número"})
	}

	mu.Lock()
	defer mu.Unlock()

	tarefa, existe := tarefas[id]
	if !existe {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tarefa não encontrada"})
	}

	return c.JSON(http.StatusOK, tarefa)
}

// apaga tarefa pelo ID
func apagarTarefa(c *echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id precisa ser um número"})
	}

	mu.Lock()
	defer mu.Unlock()

	if _, existe := tarefas[id]; !existe {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tarefa não encontrada"})
	}

	delete(tarefas, id)

	if err := salvar(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "falha ao salvar tarefas"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Tarefa apagada com sucesso!"})
}

// marca a tarefa como feita pelo ID
func marcarFeita(c *echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id precisa ser um número"})
	}

	mu.Lock()
	defer mu.Unlock()

	tarefaAtual, existe := tarefas[id]
	if !existe {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tarefa não encontrada"})
	}

	tarefaAtual.Feita = true
	tarefas[id] = tarefaAtual

	if err := salvar(); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "falha ao salvar tarefas"})
	}

	return c.JSON(http.StatusOK, tarefaAtual)
}

// busca tarefas cujo título contém o texto de ?q=, sem diferenciar maiúsculas
func buscarPorTexto(c *echo.Context) error {
	texto := c.QueryParam("q")
	if texto == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "query param 'q' é obrigatório"})
	}

	mu.Lock()
	defer mu.Unlock()

	textoMin := strings.ToLower(texto)
	lista := []Tarefa{}

	for _, t := range tarefas {
		if strings.Contains(strings.ToLower(t.Titulo), textoMin) {
			lista = append(lista, t)
		}
	}

	sort.Slice(lista, func(i, j int) bool { return lista[i].ID < lista[j].ID })

	return c.JSON(http.StatusOK, lista)
}

// salva as tarefas em tarefas.json
// chame só com o mu já trancado
func salvar() error {
	lista := []Tarefa{}
	for _, t := range tarefas {
		lista = append(lista, t)
	}

	dados, err := json.MarshalIndent(lista, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("tarefas.json", dados, 0644)
}

// lê tarefas.json quando o servidor liga
func carregar() error {
	content, err := os.ReadFile("tarefas.json")
	if errors.Is(err, os.ErrNotExist) {
		return nil // primeira vez: ainda não existe arquivo, e tudo bem
	}
	if err != nil {
		return err
	}

	var lista []Tarefa
	if err := json.Unmarshal(content, &lista); err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	for _, t := range lista {
		tarefas[t.ID] = t // coloca a tarefa de volta no map
		if t.ID >= proximoID {
			proximoID = t.ID + 1
		}
	}

	return nil
}

func main() {
	// cria o echo e configura o middleware
	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	// carrega as tarefas do arquivo; se falhar, não liga o servidor
	if err := carregar(); err != nil {
		e.Logger.Error("falha ao carregar as tarefas", "error", err)
		return
	}

	//handle e routes
	e.PUT("/tarefas/:id", func(c *echo.Context) error {
		var tarefa Tarefa

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "id precisa ser um número"})
		}

		if err := c.Bind(&tarefa); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "bad request"})
		}

		if tarefa.Titulo == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "o título é obrigatório"})
		}

		mu.Lock()
		defer mu.Unlock()

		tarefaAtual, existe := tarefas[id]
		if !existe {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "tarefa não encontrada"})
		}

		tarefaAtual.Titulo = tarefa.Titulo
		tarefas[id] = tarefaAtual

		if err := salvar(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "falha ao salvar tarefas"})
		}

		return c.JSON(http.StatusOK, tarefaAtual)
	})

	e.GET("/", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"message": "Hello, World!"})
	})

	e.GET("/ola", ola)
	e.GET("/ola/:nome", func(c *echo.Context) error {
		nome := c.Param("nome")
		return c.JSON(http.StatusOK, map[string]string{"message": "Olá, " + nome + "!"})
	})

	e.GET("/soma", calcula)

	e.GET("/tarefas", listarTarefas)
	e.POST("/tarefas", criarTarefa)
	e.GET("/tarefas/:id", buscarTarefa)
	e.DELETE("/tarefas/:id", apagarTarefa)
	e.PATCH("/tarefas/:id/feita", marcarFeita)
	e.GET("/tarefas/buscar", buscarPorTexto)

	//inicia o servidor
	if err := e.Start(":1323"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
