package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
)

type User struct {
	Name    string
	Age     int
	Comment string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	s := &http.Server{
		Addr:    ":9000",
		Handler: mux,
	}

	fmt.Println("server start port:8080")
	s.ListenAndServe()
}

func handler(w http.ResponseWriter, r *http.Request) {
	// db接続周り
	db, err := sql.Open("mysql", "root:"+os.Getenv("DB_PASSWORD")+"@tcp(127.0.0.1:3306)/rss")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	id := 1
	var name string
	err = db.QueryRow("select name from boards where id = ?", id).Scan(&name)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(name)

	//user := User{
	//	Name: name,
	//	Age: 20,
	//	Comment: "Hello",
	//}
	//tmpl := template.Must(template.ParseFiles("./views/index.html"))
	//tmpl.Execute(w, user)
	log.Printf("%+v¥n", r)
}
