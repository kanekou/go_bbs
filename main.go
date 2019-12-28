package main

import (
	session "bbs/sessions"
	"container/list"
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
	Email    string
	Password string
}

var globalSessions *session.Manager
var pder = &session.Providers{List: list.New()}
var db *sql.DB

func init() {
	pder.Sessions = make(map[string]*list.Element, 0)
	session.Register("memory", pder)
	globalSessions, _ = session.NewManager("memory", "gosessionid", 3600)
	go globalSessions.GC()
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	sess := globalSessions.SessionStart(w, r)

	switch r.Method {
	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("public/login.html"))
		w.Header().Set("Content-Type", "text/html")
		email := sess.Get("email")
		err := db.QueryRow("select email from users where email = ?", email).Scan(&email)
		if err != sql.ErrNoRows { // Empty set
			log.Println(err)
			http.Redirect(w, r, "/index", http.StatusFound)
		}

		tmpl.Execute(w, nil)
	case http.MethodPost:
		email := r.FormValue("email")
		pw := r.FormValue("password")
		err := db.QueryRow("select id from users where email = ? and password = ?", email, pw).Scan(&email)
		if err == sql.ErrNoRows { // Empty set
			log.Println(err)
			http.Redirect(w, r, "/login", http.StatusFound)
		} else {
			sess.Set("email", r.Form["email"])
			http.Redirect(w, r, "/index", http.StatusFound)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed) // 405
		w.Write([]byte("Method not allowed"))
		http.Redirect(w, r, "/index", http.StatusFound)
	}
}

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sess := globalSessions.SessionStart(w, r)
		email := sess.Get("email")

		err := db.QueryRow("select email from users where email = ?", email).Scan(&email)
		if err != sql.ErrNoRows {
			http.Redirect(w, r, "/index", http.StatusFound)
		}

		tmpl := template.Must(template.ParseFiles("public/signup.html"))
		tmpl.Execute(w, nil)
	case http.MethodPost:
		email := r.FormValue("email")
		pw := r.FormValue("password")

		insert, err := db.Prepare("insert into users(email, password) values (?,?)")
		if err != nil {
			log.Println(err)
		}
		insert.Exec(email, pw)

		http.Redirect(w, r, "/index", http.StatusFound)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed) // 405
		w.Write([]byte("Method not allowed"))
		http.Redirect(w, r, "/signup", http.StatusFound)
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	sess := globalSessions.SessionStart(w, r)

	switch r.Method {
	case http.MethodGet:
		email := sess.Get("email")
		err := db.QueryRow("select email from users where email = ?", email).Scan(&email)
		if err == sql.ErrNoRows {
			log.Println(err)
			http.Redirect(w, r, "/login", http.StatusFound)
		}
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
			if r.FormValue("logout") != "" {
				sess.Delete("email")
				http.Redirect(w, r, "login", http.StatusFound)
			} else {
				name := r.FormValue(("nickname"))
				message := r.FormValue("message")
				insert, err := db.Prepare("insert into boards(name,  message) values (?,?)")
				if err != nil {
					log.Println(err)
				}
				insert.Exec(name, message)
			}
		}

		http.Redirect(w, r, "/index", http.StatusFound)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed) // 405
		w.Write([]byte("Method not allowed"))
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
	db, err = sql.Open("mysql", "root:"+os.Getenv("DB_PASSWORD")+"@tcp(127.0.0.1:3306)/bbs")
	if err != nil {
		log.Println(err)
	}
	err = db.Ping()
	if err != nil {
		log.Println(err)
	}
	defer db.Close()

	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/signup", SignupHandler)
	http.HandleFunc("/index", IndexHandler)

	s := &http.Server{
		Addr: ":9000",
	}

	fmt.Println("server start port:8080")
	s.ListenAndServe()
}
