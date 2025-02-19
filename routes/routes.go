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
	mux.HandleFunc("/update-todo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		handlers.UpdateTodo(w, r)
	})

	mux.HandleFunc("/todo/delete/", handlers.DeleteTodo)

	return mux
}
