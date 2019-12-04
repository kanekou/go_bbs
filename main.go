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

	//mux := http.NewServeMux()
	//http.Handle("/", http.FileServer(http.Dir("public")))
	http.HandleFunc("/index", staticHandler)
	s := &http.Server{
		Addr: ":9000",
		//Handler: mux,
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

	// クエリ発行
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
		fmt.Println(user)
		users = append(users, user)
	}

	tmpl := template.Must(template.ParseFiles("public/index.html"))
	tmpl.Execute(w, users)
	log.Printf("%+v¥n", r)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	db := dbConnect()
	defer db.Close()

	// Formデータを取得.
	form := r.PostForm
	fmt.Fprintf(w, "フォーム：\n%v\n", form)

	// または、クエリパラメータも含めて全部.
	params := r.Form
	fmt.Fprintf(w, "フォーム2：\n%v\n", params)
}
