package routes

import (
	"net/http"
	"todo-api/handlers"
)

func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/todos", handlers.GetTodos)
	mux.HandleFunc("/todo", handlers.GetTodoByID)
	mux.HandleFunc("/todo/create", handlers.CreateTodo)
	mux.HandleFunc("/todoss",handlers.GetTodosWithFilterSortPagination)
	mux.HandleFunc("/todo/update/", handlers.UpdateTodo)
	mux.HandleFunc("/todo/delete/", handlers.DeleteTodo)

	return mux
}
