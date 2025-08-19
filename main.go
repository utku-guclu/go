package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/lib/pq"
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
	rdb.set(ctx, taskCacheKey, taskCacheData, 0)

	// Send the created task as a JSON response
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}
