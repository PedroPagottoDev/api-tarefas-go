package main

import (
	"net/http"
	"sort"
	"strconv"

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

	tarefa.ID = proximoID // o ID é gerado por nós, não vem de fora
	proximoID++
	tarefas[tarefa.ID] = tarefa

	return c.JSON(http.StatusCreated, tarefa)
}

// busca uma tarefa pelo ID
func buscarTarefa(c *echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id precisa ser um número"})
	}

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

	if _, existe := tarefas[id]; !existe {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tarefa não encontrada"})
	}

	delete(tarefas, id)
	return c.JSON(http.StatusOK, map[string]string{"message": "Tarefa apagada com sucesso!"})
}

// marca se a tarefa foi feita pelo ID
func marcarFeita(c *echo.Context) error {

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id precisa ser um número"})
	}

	tarefaAtual, existe := tarefas[id]

	if !existe {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "tarefa não encontrada"})
	}

	tarefaAtual.Feita = true
	tarefas[id] = tarefaAtual

	return c.JSON(http.StatusOK, tarefaAtual)

}

func main() {
	// cria o echo e configura o middleware
	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	//handle e routes
	e.PUT("/tarefas/:id", func(c *echo.Context) error {
		var tarefa Tarefa

		id, err := strconv.Atoi(c.Param("id"))

		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "id precisa ser um número"})
		}

		tarefaAtual, existe := tarefas[id]
		if !existe {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "tarefa não encontrada"})
		}

		if err := c.Bind(&tarefa); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "bad request"})
		}

		if tarefa.Titulo == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "o título é obrigatório"})
		}

		tarefaAtual.Titulo = tarefa.Titulo
		tarefas[id] = tarefaAtual

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

	//inicia o servidor
	if err := e.Start(":1323"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
