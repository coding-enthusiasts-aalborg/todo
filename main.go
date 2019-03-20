/* This is a quick mockup of how a todo app could work in Go.
 * The website part is not finished yet, but you can create new todos.
 */

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/boltdb/bolt"
)

func main() {
	// Setup the database
	bucketName := "todo-bucket"
	dbName := "todo.db"
	port := 8080

	db, err := bolt.Open("todo.db", 0600, nil)
	if err != nil {
		exit(fmt.Sprintf("could not open database file: %s", dbName))
	}

	// Close the database whenever the program closes, i.e. this function returns.
	defer db.Close()

	// A bucket is a place where you put your data. The data can be of any kind
	// quite dissimilar to SQL.
	ensureBucketCreated(db, bucketName)

	// Load in the template
	templatePath := "form.html"
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		exit(fmt.Sprintf("could not load or parse template file: %s", templatePath))
	}

	// Create a "handler", which is a struct (like a class)
	// which has some data needed for serving the websites
	h := handler{db, bucketName, tmpl}

	fmt.Printf("Running server on http://localhost:%d\n", port)

	// Run the server on port 8080
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), h))
}

type handler struct {
	db         *bolt.DB
	bucketName string
	tmpl       *template.Template
}

// This is a method, because it works on some struct, here the handler.
// The difference is seen between the word "func" and the name of the function "ServeHTTP".
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	nameOfTodo := r.FormValue("todo-name")

	// Handling of the different "HTTP verbs"
	switch r.Method {
	case "GET":
		h.showTodos(w)
	case "POST":
		h.createNew(nameOfTodo)
		h.showTodos(w)
	case "PUT":
		h.updateTodo(nameOfTodo)
		h.showTodos(w)
	case "DELETE":
		h.delete(nameOfTodo)
		h.showTodos(w)
	}
}

func (h handler) showTodos(w http.ResponseWriter) {
	h.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(h.bucketName))

		todos := make(map[string]bool)

		// This turned out be a bit tedious because BoltDB doesn't support bools.
		// So it converts a string to a bool.
		b.ForEach(func(k, v []byte) error {
			name := string(k)
			val, err := strconv.ParseBool(string(v))
			if err != nil {
				return err
			}
			todos[name] = val
			return nil
		})
		err := h.tmpl.Execute(w, todos)
		if err != nil {
			panic("Could not execute")
		}
		return err
	})
}

// The content of the database buckets are arrays of bytes.
// We convert the string false into bytes.
func (h handler) createNew(nameOfTodo string) {
	h.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(h.bucketName))
		err := b.Put([]byte(nameOfTodo), []byte("false"))
		return err
	})
}

// Takes out the current value and flips it.
func (h handler) updateTodo(nameOfTodo string) {
	h.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(h.bucketName))
		v := b.Get([]byte(nameOfTodo))
		if v == nil {
			return fmt.Errorf("could not find item with key: %s", nameOfTodo)
		}
		var newVal string
		if string(v) == "false" {
			newVal = "true"
		} else {
			newVal = "false"
		}
		err := b.Put([]byte(nameOfTodo), []byte(newVal))
		return err
	})
}

func (h handler) delete(nameOfTodo string) {
	h.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(h.bucketName))
		err := b.Delete([]byte(nameOfTodo))
		return err
	})
}

// The buckets need to be created before usage.
func ensureBucketCreated(db *bolt.DB, bucketName string) {
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketName))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
}

func exit(msg string) {
	println(msg)
	os.Exit(1)
}
