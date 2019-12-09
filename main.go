package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"net/http"
	"os"
)

type Board struct {
	Id      int
	Name    string
	Message string
}

type User struct {
	Id       int
	Name     string
	Email    string
	Password string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/index", staticHandler)
	s := &http.Server{
		Addr: ":9000",
	}

	fmt.Println("server start port:8080")
	s.ListenAndServe()
}

func dbConnect() (db *sql.DB) {
	db, err := sql.Open("mysql", "root:"+os.Getenv("DB_PASSWORD")+"@tcp(127.0.0.1:3306)/bbs")
	if err != nil {
		log.Println(err)
	}
	return db
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("public/login.html"))
		tmpl.Execute(w, nil)
	case http.MethodPost:
		var id string
		email := r.FormValue("email")
		pw := r.FormValue("password")

		db := dbConnect()
		defer db.Close()
		err := db.QueryRow("select id from users where email = ? and password = ?", email, pw).Scan(&id)
		if err == sql.ErrNoRows { // Empty set
			log.Println(err)
			http.Redirect(w, r, "/login", http.StatusFound)
		}
		fmt.Println(id)
		http.Redirect(w, r, "/index", http.StatusFound)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed) // 405
		w.Write([]byte("Method not allowed"))
		http.Redirect(w, r, "/index", http.StatusFound)
	}
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	db := dbConnect()
	defer db.Close()

	switch r.Method {
	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("public/signup.html"))
		tmpl.Execute(w, nil)
	case http.MethodPost:
		name := r.FormValue("name")
		email := r.FormValue("email")
		pw := r.FormValue("password")

		db := dbConnect()
		defer db.Close()
		insert, err := db.Prepare("insert into users(name, email, password) values (?,?,?)")
		if err != nil {
			log.Println(err)
		}
		insert.Exec(name, email, pw)

		http.Redirect(w, r, "/index", http.StatusFound)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed) // 405
		w.Write([]byte("Method not allowed"))
		http.Redirect(w, r, "/signup", http.StatusFound)
	}
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	db := dbConnect()
	defer db.Close()

	switch r.Method {
	case http.MethodGet:
		result, err := db.Query("select * from boards")
		if err != nil {
			log.Println(err)
		}
		defer result.Close()

		boards := make([]Board, 0)
		for result.Next() {
			var board Board
			if err := result.Scan(&board.Id, &board.Name, &board.Message); err != nil {
				log.Println(err)
			}
			boards = append(boards, board)
		}

		tmpl := template.Must(template.ParseFiles("public/index.html"))
		tmpl.Execute(w, boards)
		log.Printf("%+v¥n", r)
	case http.MethodPost:
		method := r.PostFormValue("_method")
		if method == "DELETE" {
			id := r.PostFormValue("id")
			delete, err := db.Prepare("delete from boards where id = ?")
			if err != nil {
				log.Println(err)
			}
			delete.Exec(id)
		} else { //post
			name := r.FormValue(("nickname"))
			message := r.FormValue("message")
			insert, err := db.Prepare("insert into boards(name,  message) values (?,?)")
			if err != nil {
				log.Println(err)
			}
			insert.Exec(name, message)
		}

		http.Redirect(w, r, "/index", http.StatusFound)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed) // 405
		w.Write([]byte("Method not allowed"))
	}
}
