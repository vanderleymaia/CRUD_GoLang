package main

//TO USE THIS TEMPLATE YOU HAVE TO CONFIGURE A MySQL SERVER.
//YOU CAN USE DOCKER FOR THIS

//TO RUN A DOCKER
//docker run -d -p 3306:3306 --name mysql -e MYSQL_ROOT_PASSWORD=root mysql

//TO CONFIGURE YOUR DATABASE
//$ docker exec -it mysql mysql -uroot -proot -e 'CREATE DATABASE todolist'

import (
	"encoding/json"
	"fmt"
	"io"

	"net/http"

	log "github.com/sirupsen/logrus"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/rs/cors"

	"strconv"
)

var db, _ = gorm.Open("mysql", "root:root@/todolist?charset=utf8&parseTime=True&loc=Local")

// TodoItemModel is the model for the tasks
type TodoItemModel struct {
	ID          int `gorm:"primary_key"`
	Description string
	Location    string
	Completed   bool
}

// Healthz was writen to allow the user check if the api is alive
func Healthz(w http.ResponseWriter, r *http.Request) {
	log.Info("API Health is OK")
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"alive": true}`)
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetReportCaller(true)
}

func main() {
	defer db.Close()

	//db.Debug().DropTableIfExists(&TodoItemModel{})
	db.Debug().AutoMigrate(&TodoItemModel{})

	log.Info("Starting Todolist API server")
	router := mux.NewRouter()
	router.HandleFunc("/healthz", Healthz).Methods("GET")
	router.HandleFunc("/todo", CreateItem).Methods("POST")
	router.HandleFunc("/todo2", CreateItem2).Methods("POST")

	router.HandleFunc("/todo/{id}", UpdateItem).Methods("POST")
	router.HandleFunc("/todo2/{id}", UpdateItem2).Methods("POST")
	router.HandleFunc("/todo/{id}", DeleteItem).Methods("DELETE")
	router.HandleFunc("/todo-completed", GetCompletedItems).Methods("GET")
	router.HandleFunc("/todo-incomplete", GetIncompleteItems).Methods("GET")

	handler := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "DELETE", "PATCH", "OPTIONS"},
	}).Handler(router)

	http.ListenAndServe(":8000", handler)
}

// CreateItem is responsible to create new itens
func CreateItem(w http.ResponseWriter, r *http.Request) {
	description := r.FormValue("description")
	location := r.FormValue("location")

	log.WithFields(log.Fields{
		"description": description,
		"location":    location}).Info("Add new TodoItem. Saving to database.")
	todo := &TodoItemModel{Description: description, Location: location, Completed: false}
	db.Create(&todo)
	result := db.Last(&todo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Value)
}

// CreateItem2 is responsible to create new itens (USING BODY JSON DATA)
func CreateItem2(w http.ResponseWriter, r *http.Request) {
	//Declare a new task
	var task TodoItemModel

	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	todo := &TodoItemModel{Description: task.Description, Location: task.Location, Completed: false}
	db.Create(&todo)
	result := db.Last(&todo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Value)
}

// UpdateItem is responsible to update new itens
func UpdateItem(w http.ResponseWriter, r *http.Request) {
	// Get URL parameter from mux
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	// Test if the TodoItem exist in DB
	err := GetItemByID(id)
	if err == false {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"updated": false, "error": "Record Not Found"}`)
	} else {
		completed, _ := strconv.ParseBool(r.FormValue("completed"))
		log.WithFields(log.Fields{"Id": id, "Completed": completed}).Info("Updating TodoItem")
		todo := &TodoItemModel{}
		db.First(&todo, id)
		todo.Completed = completed
		db.Save(&todo)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"updated": true}`)
	}
}

// UpdateItem2 is responsible to update new itens (USING BODY JSON DATA)
func UpdateItem2(w http.ResponseWriter, r *http.Request) {

	fmt.Printf("Updating...")
	// Get URL parameter from mux
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	fmt.Printf("ID %v", id)

	//Declare a new task
	var task TodoItemModel

	errDec := json.NewDecoder(r.Body).Decode(&task)
	if errDec != nil {
		http.Error(w, errDec.Error(), http.StatusBadRequest)
	}

	fmt.Println(task)

	// Test if the TodoItem exist in DB
	err := GetItemByID(id)
	if err == false {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"updated": false, "error": "Record Not Found"}`)
	} else {
		var todo TodoItemModel
		db.First(&todo, id)

		todo.Completed = task.Completed
		todo.Description = task.Description
		todo.Location = task.Location

		fmt.Println(todo)
		db.Save(&todo)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"updated": true}`)
	}
}

// DeleteItem is responsible to delete itens
func DeleteItem(w http.ResponseWriter, r *http.Request) {
	// Get URL parameter from mux
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	// Test if the TodoItem exist in DB
	err := GetItemByID(id)
	if err == false {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"deleted": false, "error": "Record Not Found"}`)
	} else {
		log.WithFields(log.Fields{"Id": id}).Info("Deleting TodoItem")
		todo := &TodoItemModel{}
		db.First(&todo, id)
		db.Delete(&todo)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"deleted": true}`)
	}
}

// GetItemByID is responsible to return an specific item
func GetItemByID(ID int) bool {
	todo := &TodoItemModel{}
	result := db.First(&todo, ID)
	if result.Error != nil {
		log.Warn("TodoItem not found in database")
		return false
	}
	return true
}

// GetCompletedItems is responsible to return all completed itens
func GetCompletedItems(w http.ResponseWriter, r *http.Request) {
	log.Info("Get completed TodoItems")
	completedTodoItems := GetTodoItems(true)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(completedTodoItems)
}

// GetIncompleteItems is responsible to return all not completed itens
func GetIncompleteItems(w http.ResponseWriter, r *http.Request) {
	log.Info("Get Incomplete TodoItems")
	IncompleteTodoItems := GetTodoItems(false)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(IncompleteTodoItems)
}

// GetTodoItems is responsible to return all  itens
func GetTodoItems(completed bool) interface{} {
	var todos []TodoItemModel
	TodoItems := db.Where("completed = ?", completed).Find(&todos).Value
	return TodoItems
}
