package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"todo-api/database"
	"todo-api/models"

	"github.com/google/uuid"
)

var db *sql.DB

func init() {

	db = database.ConnectDB()
	if db == nil {
		log.Fatalf("Failed to connect to the database")
	}
}

func LogAction(action string, todoID uuid.UUID, details string, message string) {
	// Log the action performed (e.g., create, update, delete)
	query := `INSERT INTO logs (action, todo_id, message, details, timestamp) VALUES ($1, $2, $3, $4, NOW())`
	_, err := db.Exec(query, action, todoID, message, details)
	if err != nil {
		log.Printf("Failed to log action: %v", err)
	}
}
func CreateTodo(w http.ResponseWriter, r *http.Request) {
	var todo models.Todo

	// Decode the JSON request body
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		log.Printf("Error decoding request body: %v", err) // Log the error for debugging
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		LogAction("create", uuid.Nil, "Invalid request payload", "Failed to decode JSON request body") // Log the error with nil UUID
		return
	}

	// Generate a new UUID for the Todo item
	todo.ID = uuid.New()
	fmt.Println("Generated UUID for todo:", todo.ID) // Debugging: Log the generated UUID

	// Convert DueDate to string (YYYY-MM-DD) format
	dueDateStr := todo.DueDate.Format("2006-01-02")

	// Insert the todo into the database and return the inserted ID
	query := `INSERT INTO todos (id, title, description, status, due_date, created_at, is_deleted)
	          VALUES ($1, $2, $3, $4, $5, NOW(), FALSE) RETURNING id`
	err := db.QueryRow(query, todo.ID, todo.Title, todo.Description, todo.Status, dueDateStr).Scan(&todo.ID)
	if err != nil {
		log.Printf("Error creating todo: %v", err) // Log error for debugging
		http.Error(w, "Failed to create todo", http.StatusInternalServerError)
		LogAction("create", uuid.Nil, "Failed to create todo", fmt.Sprintf("Error: %v", err)) // Log failure with nil UUID
		return
	}

	fmt.Println("Final inserted UUID in database:", todo.ID) // Debugging: Log the returned UUID

	// Log the action with the correct UUID
	LogAction("create", todo.ID, "Todo created", "Creation of new todo item")

	// Respond with created todo
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(todo)
}

func GetTodos(w http.ResponseWriter, r *http.Request) {
	// Fetch all todos without filtering or pagination
	rows, err := db.Query("SELECT id, title, status,due_date,description FROM todos WHERE is_deleted = FALSE")
	if err != nil {
		http.Error(w, "Unable to fetch todos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var todos []models.Todo
	for rows.Next() {
		var todo models.Todo
		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Status, &todo.DueDate, &todo.Description); err != nil {
			http.Error(w, "Unable to read todo", http.StatusInternalServerError)
			return
		}
		todos = append(todos, todo)
	}

	// If no todos are found, return a 404 status
	if len(todos) == 0 {
		response := map[string]interface{}{
			"status":        "404 Not Found",
			"message":       "No todos found",
			"total_todos":   0,
			"server_status": "OK",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Prepare the response with the status, todos, and total count
	response := map[string]interface{}{
		"status":        "200 OK",
		"todos":         todos,
		"total_todos":   len(todos),
		"server_status": "OK",
	}

	// Set Content-Type header to JSON
	w.Header().Set("Content-Type", "application/json")

	// Return the JSON response with 200 OK status
	w.WriteHeader(http.StatusOK) // Set the status code to 200 OK
	json.NewEncoder(w).Encode(response)
}

// GetTodosWithFilterSortPagination retrieves todos with filtering, sorting, and pagination
func GetTodosWithFilterSortPagination(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	queryParams := r.URL.Query()

	page, err := strconv.Atoi(queryParams.Get("page"))
	if err != nil || page < 1 {
		page = 1 // Default page = 1
	}

	limit, err := strconv.Atoi(queryParams.Get("limit"))
	if err != nil || limit < 1 {
		limit = 10 // Default limit = 10
	}

	status := queryParams.Get("status")
	dueDate := queryParams.Get("due_date")
	sortBy := queryParams.Get("sort_by")
	sortOrder := queryParams.Get("sort_order")

	// Validate sorting parameters
	allowedSortFields := map[string]bool{"id": true, "title": true, "status": true, "due_date": true, "created_at": true}
	if !allowedSortFields[sortBy] {
		sortBy = "created_at" // Default sort field
	}
	if sortOrder != "ASC" {
		sortOrder = "DESC" // Default sort order
	}

	offset := (page - 1) * limit

	// **Prepare SQL Query with placeholders**
	query := "SELECT id, title, description, status, due_date, created_at, is_deleted FROM todos WHERE is_deleted = false"
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}
	if dueDate != "" {
		query += fmt.Sprintf(" AND due_date = $%d", argIndex)
		args = append(args, dueDate)
		argIndex++
	}

	// **Sorting**
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// **Pagination**
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	// **Execute Query**
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to fetch todos: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var todos []models.Todo
	for rows.Next() {
		var todo models.Todo
		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Status, &todo.DueDate, &todo.CreatedAt, &todo.IsDeleted); err != nil {
			http.Error(w, "Unable to parse todo data", http.StatusInternalServerError)
			return
		}
		todos = append(todos, todo)
	}

	// **Count total todos for pagination**
	countQuery := "SELECT COUNT(*) FROM todos WHERE is_deleted = false"
	countArgs := []interface{}{}
	countIndex := 1

	if status != "" {
		countQuery += fmt.Sprintf(" AND status = $%d", countIndex)
		countArgs = append(countArgs, status)
		countIndex++
	}
	if dueDate != "" {
		countQuery += fmt.Sprintf(" AND due_date = $%d", countIndex)
		countArgs = append(countArgs, dueDate)
		countIndex++
	}

	var totalTodos int
	err = db.QueryRow(countQuery, countArgs...).Scan(&totalTodos)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to count todos: %v", err), http.StatusInternalServerError)
		return
	}

	// **Calculate total pages**
	totalPages := (totalTodos + limit - 1) / limit

	// **Prepare JSON Response**
	response := map[string]interface{}{
		"status":       200,
		"todos":        todos,
		"current_page": page,
		"total_pages":  totalPages,
		"total_todos":  totalTodos,
	}

	// **Set Response Headers and Send Response**
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
// GetTodoByID retrieves a specific todo by ID
func GetTodoByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		status := http.StatusBadRequest
		response := map[string]interface{}{
			"status":  status,
			"message": "Missing ID parameter",
		}
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)

		LogAction("fetch", uuid.Nil, "Missing ID parameter", fmt.Sprintf("Status: %d - Missing ID in the request", status))
		return
	}

	// Parse the ID into a UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		status := http.StatusBadRequest
		response := map[string]interface{}{
			"status":  status,
			"message": "Invalid ID format",
		}
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)

		LogAction("fetch", uuid.Nil, "Invalid ID format", fmt.Sprintf("Status: %d - Invalid UUID format", status))
		return
	}

	var todo models.Todo
	query := "SELECT id, title, description, status, due_date, created_at, is_deleted FROM todos WHERE id = $1 AND is_deleted = false"
	err = db.QueryRow(query, id).Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Status, &todo.DueDate, &todo.CreatedAt, &todo.IsDeleted)

	if err != nil {
		if err == sql.ErrNoRows {
			status := http.StatusNotFound
			response := map[string]interface{}{
				"status":  status,
				"message": "Todo not found",
			}
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(response)

			LogAction("fetch", id, "Todo not found", fmt.Sprintf("Status: %d - Todo with the given ID does not exist", status))
		} else {
			status := http.StatusInternalServerError
			response := map[string]interface{}{
				"status":  status,
				"message": "Failed to fetch todo",
				"error":   err.Error(),
			}
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(response)

			LogAction("fetch", id, "Failed to fetch todo", fmt.Sprintf("Status: %d - Error: %v", status, err))
		}
		return
	}

	// Successful response
	status := http.StatusOK
	response := map[string]interface{}{
		"status":  status,
		"message": "Todo fetched successfully",
		"data":    todo,
	}
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)

	LogAction("fetch", todo.ID, "Todo fetched successfully", fmt.Sprintf("Status: %d - Todo: %v", status, todo.Title))
}
// UpdateTodo updates a todo item and logs both previous and updated values.
func UpdateTodo(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		log.Println("[ERROR] Missing todo ID")
		http.Error(w, "Missing todo ID", http.StatusBadRequest)
		return
	}

	log.Printf("[INFO] Received PUT request to update Todo ID: %s\n", id)

	// Step 1: Fetch previous todo details before updating
	var prevTodo models.Todo
	prevQuery := "SELECT id, title, description, status, due_date FROM todos WHERE id = $1"
err := db.QueryRow(prevQuery, id).Scan(&prevTodo.ID, &prevTodo.Title, &prevTodo.Description, &prevTodo.Status, &prevTodo.DueDate)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("[WARN] Todo ID: %s not found\n", id)
			http.Error(w, "Todo not found", http.StatusNotFound)
		} else {
			log.Printf("[ERROR] Failed to fetch previous todo data: %v\n", err)
			http.Error(w, "Failed to fetch previous todo data", http.StatusInternalServerError)
		}
		return
	}

	// Step 2: Decode the new update request
	var newTodo models.Todo
	if err := json.NewDecoder(r.Body).Decode(&newTodo); err != nil {
		log.Printf("[ERROR] Invalid request payload: %v\n", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	log.Printf("[DEBUG] Parsed JSON Body: %+v\n", newTodo)

	// Step 3: Prepare the update query dynamically
	query := "UPDATE todos SET "
	var values []interface{}
	var setClauses []string
	paramIndex := 1

	if newTodo.Title != "" {
		setClauses = append(setClauses, fmt.Sprintf("title=$%d", paramIndex))
		values = append(values, newTodo.Title)
		paramIndex++
	}
	if newTodo.Description != "" {
		setClauses = append(setClauses, fmt.Sprintf("description=$%d", paramIndex))
		values = append(values, newTodo.Description)
		paramIndex++
	}
	if newTodo.Status != "" {
		setClauses = append(setClauses, fmt.Sprintf("status=$%d", paramIndex))
		values = append(values, newTodo.Status)
		paramIndex++
	}

	if len(setClauses) == 0 {
		log.Println("[WARN] No fields provided for update")
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}
	if !newTodo.DueDate.IsZero() { // Correct way to check if DueDate is set
		setClauses = append(setClauses, fmt.Sprintf("due_date=$%d", paramIndex))
		values = append(values, newTodo.DueDate)
		paramIndex++
	}
	
	

	query += strings.Join(setClauses, ", ") + fmt.Sprintf(" WHERE id=$%d RETURNING id, title, description, status, due_date", paramIndex)
	values = append(values, id)

	// Step 4: Execute the update query and fetch new data
	var updatedTodo models.Todo
	err = db.QueryRow(query, values...).Scan(&updatedTodo.ID, &updatedTodo.Title, &updatedTodo.Description, &updatedTodo.Status, &updatedTodo.DueDate)
	if err != nil {
		log.Printf("[ERROR] Failed to update todo: %v\n", err)
		http.Error(w, "Failed to update todo", http.StatusInternalServerError)
		return
	}

	// Step 5: Log the update action with both previous and new data
	logMessage := fmt.Sprintf(
		"Todo Updated: [Prev] Title: %s, Description: %s, Status: %s → [New] Title: %s, Description: %s, Status: %s",
		prevTodo.Title, prevTodo.Description, prevTodo.Status,
		updatedTodo.Title, updatedTodo.Description, updatedTodo.Status,
	)
	LogAction("update", updatedTodo.ID, "Todo updated successfully", logMessage)

	log.Printf("[INFO] Todo ID: %s updated successfully.\n", updatedTodo.ID)

	// Step 6: Return the response including both previous and updated values
	response := map[string]interface{}{
		"message":  "Todo updated successfully",
		"previous": prevTodo,
		"updated":  updatedTodo,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}



func DeleteTodo(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		response := map[string]interface{}{
			"status":  "error",
			"message": "Missing ID parameter",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		LogAction("delete", uuid.Nil, "Missing ID parameter", "Failed to delete todo: No ID provided")
		return
	}

	// Parse the UUID
	id, err := uuid.Parse(idStr)
	if err != nil {
		response := map[string]interface{}{
			"status":  "error",
			"message": "Invalid ID format",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		LogAction("delete", uuid.Nil, "Invalid ID format", "Failed to delete todo: Invalid UUID format")
		return
	}

	// Fetch the todo details before deleting
	var todo struct {
		ID          uuid.UUID `json:"id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Status      string    `json:"status"`
		DueDate     time.Time `json:"due_date"`
	}

	err = db.QueryRow("SELECT id, title, description, status, due_date FROM todos WHERE id = $1 AND is_deleted = FALSE", id).
		Scan(&todo.ID, &todo.Title, &todo.Description, &todo.Status, &todo.DueDate)

	if err != nil {
		if err == sql.ErrNoRows {
			response := map[string]interface{}{
				"status":  "error",
				"message": "Todo not found or already deleted",
			}
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response)
			LogAction("delete", id, "Todo not found", "Failed to delete todo: Already deleted or does not exist")
			return
		}

		response := map[string]interface{}{
			"status":  "error",
			"message": "Database error",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		LogAction("delete", id, "Database error", fmt.Sprintf("Failed to fetch todo: %v", err))
		return
	}

	// Perform a soft delete
	query := "UPDATE todos SET is_deleted = TRUE WHERE id = $1"
	_, err = db.Exec(query, id)
	if err != nil {
		response := map[string]interface{}{
			"status":  "error",
			"message": "Failed to delete todo",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		LogAction("delete", id, "Failed to delete", fmt.Sprintf("Error executing delete query: %v", err))
		return
	}

	LogAction("delete", id, "Todo deleted", "Successfully marked todo as deleted")

	// Return success response with deleted todo details
	response := map[string]interface{}{
		"status":  "success",
		"message": "Todo deleted successfully",
		"todo":    todo,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
