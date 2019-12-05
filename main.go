package main

import (
	"database/sql"
	"fmt"
	"github.com/go-delve/delve/service/api"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"net/http"
	"os"
)

type User struct {
	Id      int
	Name    string
	Email   string
	Message string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	http.HandleFunc("/login", loginHandler)
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
		log.Fatal(err)
	}
	return db
}

//func checkAuth(r *http.Request) bool {
//	user, pw, ok := r.BasicAuth()
//	if !ok || user != autuUser || pw != authPw {
//		return false
//	}
//	return true
//}
//
//func basicAuthHandler(w http.ResponseWriter, r *http.Request) {
//	if checkAuth(r) == false {
//		w.Header().Add("WWW.Authenticate", `Basic realm="my private area"`)
//		w.WriteHeader(http.StatusUnauthorized)
//		w.Write([]byte("401 Not Authenticate"))
//		return
//	}
//
//	tmpl := template.Must(template.ParseFiles("public/login.html"))
//	tmpl.Execute(w, )
//}

//func Secret(user, realm string) string {
//	db := dbConnect()
//	defer db.Close()
//	login, err := db.Prepare("select * from users where email = ? and password = ?")
//	if err != nil {
//		log.Fatal(err)
//	}
//	login.Exec(user, email)
//}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		http.FileServer(http.Dir("public/login.html"))
	case http.MethodPost:
		//TODO: email, pw をparamsから持ってくる
		db := dbConnect()
		defer db.Close()
		login, err := db.Prepare("select * from users where email = ? and password = ?")
		if err != nil {
			log.Fatal(err)
			http.Redirect(w, r, "/login", http.StatusFound)
		}
		login.Exec(email, pw)
		http.Redirect(w, r, "/index", http.StatusFound)
	default:
		fmt.Fprint(w, "Method not allowed.\n")
	}
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	db := dbConnect()
	defer db.Close()

	switch r.Method {
	case http.MethodGet:
		result, err := db.Query("select * from boards")
		if err != nil {
			log.Fatal(err)
		}
		defer result.Close()

		users := make([]User, 0)
		for result.Next() {
			var user User
			if err := result.Scan(&user.Id, &user.Name, &user.Email, &user.Message); err != nil {
				log.Fatal(err)
			}
			users = append(users, user)
		}

		tmpl := template.Must(template.ParseFiles("public/index.html"))
		tmpl.Execute(w, users)
		log.Printf("%+v¥n", r)
	case http.MethodPost:
		method := r.PostFormValue("_method")
		if method == "DELETE" {
			id := r.PostFormValue("id")
			delete, err := db.Prepare("delete from boards where id = ?")
			if err != nil {
				log.Fatal(err)
			}
			delete.Exec(id)
		} else { //post
			name := r.FormValue(("name"))
			email := r.FormValue("email")
			message := r.FormValue("message")
			insert, err := db.Prepare("insert into boards(name, email, message) values (?,?,?)")
			if err != nil {
				log.Fatal(err)
			}
			insert.Exec(name, email, message)
		}

		http.Redirect(w, r, "/index", http.StatusFound)
	default:
		fmt.Fprint(w, "Method not allowed.\n")
	}
}
