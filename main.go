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
	"strings"
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

	http.HandleFunc("/index", staticHandler)
	http.HandleFunc("/post", postHandler)
	s := &http.Server{
		Addr: ":9000",
	}

	fmt.Println("server start port:8080")
	s.ListenAndServe()
}

func dbConnect() (db *sql.DB) {
	db, err := sql.Open("mysql", "root:"+os.Getenv("DB_PASSWORD")+"@tcp(127.0.0.1:3306)/rss")
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	db := dbConnect()
	defer db.Close()

	//get
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

	//post
	name := r.FormValue(("name"))
	email := r.FormValue("email")
	message := r.FormValue("message")
	insert, err := db.Prepare(" insert into boards(name, email, message) values (?,?,?)")
	if err != nil {
		log.Fatal(err)
	}
	insert.Exec(name, email, message)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	db := dbConnect()
	defer db.Close()

	r.ParseForm()

	// Formデータを取得.
	//fmt.Fprintf(w, "フォーム：\n%v\n", form)

	//name := r.Form.Get("name")
	//email := r.Form.Get("email")
	//message := r.Form.Get("message")

	r.ParseForm()
	fmt.Println(r.Form)
	fmt.Println("path", r.URL.Path)
	fmt.Println("scheme", r.URL.Scheme)
	fmt.Println(r.Form["url_long"])
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
	}
	fmt.Fprintf(w, "Hello astaxie!")

	//if len(form["name"][0]) == 0 || len(form["email"][0]) == 0 || len(form["message"][0] == 0 {
	//	log.Fatal("値を入力してください")
	//}
}
