package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"log"
	"net/http"
)

// Redis setup
var rdb *redis.Client
var ctx = context.Background()

// Database setup
var db *sql.DB

// Connect to Redis
func initRedis() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // Redis server address
		Password: "",               // No password by default
		DB:       0,                // Default DB
	})
}

// Connect PostgreSQL
func initDB() {
	var err error
	// Replace with your PostgreSQL username and password
	db, err = sql.Open("postgres", "postgres://username:password@localhost:5432/taskdb?sslmode=disable")

	if err != nil {
		log.Fatal(err)
	}

}

// Create table in PostgreSQL if it doesn't exist
func createTable() {
	query := `CREATE TABLE IF NOT EXISTS tasks (
		id SERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		completed BOOLEAN NOT NULL DEFAULT false
	);
`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

// Task structure to define the task
type Task struct {
	ID        int    `json:"id"`
	NAME      string `json:"name"`
	Completed bool   `json:"completed"`
}

// Handler to create a new task
func createTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	// Decode the incoming JSON into a Task struct
	err := json.NewDecoder(r.Body).Decode(&task)

	if err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	/*
		This code runs an INSERT SQL query with parameters, inserting a task into the tasks table.
		It then retrieves the id of the newly inserted task and stores it in task.ID.
		If the query fails, it captures the error in the err variable for error handling.
	*/
	// Insert the task into the PostgreSQL database
	query := `INSERT INTO tasks (name, completed) VALUES ($1, $2) RETURNING id`
	err = db.QueryRow(query, task.NAME, task.Completed).Scan(&task.ID)

	if err != nil {
		http.Error(w, "Error inserting task into database", http.StatusInternalServerError)
		return
	}

	// Cache the task in redis
	taskCacheKey := fmt.Sprintf("task:%d", task.ID)
	taskCacheData, _ := json.Marshal(task)
	rdb.Set(ctx, taskCacheKey, taskCacheData, 0)

	// Send the created task as a JSON response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

// Handler to get a task by ID
func getTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.URL.Query().Get("id")

	// Check Redis for cached task
	taskCacheKey := fmt.Sprintf("task:%s", taskID)
	cachedTask, err := rdb.Get(ctx, taskCacheKey).Result()

	if err == nil {
		// If the task is in Redis, return it directly
		w.Header().Set("Content-Type", "application/json")
		/* It sends the content (in this case, the byte slice representation of cachedTask) as the body of the HTTP response.
		The client (the user’s browser or any other system making the request) will receive this data.
		The content type (e.g., JSON, HTML, plain text) is usually set by setting the appropriate response header, such as Content-Type: application/json. */
		w.Write([]byte(cachedTask))
		return
	}

	// If not in Redis, fetch the task from PostgreSQL
	query := `SELECT id, name, completed FROM tasks WHERE id = $1`
	var task Task
	err = db.QueryRow(query, taskID).Scan(&task.ID, &task.NAME, &task.Completed)

	if err != nil {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	// Cache the task in Redis for future requests
	/* 	json.Marshal(task): Converts a Go object (struct) into a JSON-encoded byte slice to store in Redis. */
	taskCacheData, _ := json.Marshal(task)
	rdb.Set(ctx, taskCacheKey, taskCacheData, 0)

	// Send the task as a JSON response
	w.Header().Set("Content-Type", "application/json")
	/* 	json.NewEncoder(w).Encode(task): Converts a Go object (struct) into JSON and writes it directly to the HTTP response body. */
	json.NewEncoder(w).Encode(task)
}

func main() {
	// Initialize Redis and PostgreSQL
	initRedis()
	initDB()
	createTable()

	/* http.HandleFunc(): This binds the routes (/tasks and /task) to their respective handler functions (createTask and getTask).
	http.ListenAndServe(":8080", nil): This starts the HTTP server on port 8080. If there’s any error, it will be logged. */

	// Define API routes
	http.HandleFunc("/tasks", createTask) // POST / tasks
	http.HandleFunc("/task", getTask)     // GET / task?id=<taskID>

	// Start the server
	fmt.Println("Server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
