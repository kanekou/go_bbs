package main

import (
	"container/list"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
	//"bbs/modules"
)

type Board struct {
	Id      int
	Name    string
	Message string
}

type User struct {
	Id       int
	Name     string // 現時点では機能的に不要、sessionを実装する場合に使用する
	Email    string
	Password string
}

//session manager
type Manager struct {
	cookieName  string
	lock        sync.Mutex
	provider    Provider
	maxlifetime int64
}

type Provider interface {
	SessionInit(sid string) (Session, error)
	SessionRead(sid string) (Session, error)
	SessionDestroy(sid string) error
	SessionGC(maxLifeTime int64)
}

type Session interface {
	Set(key, value interface{}) error
	Get(key interface{}) interface{}
	Delete(key interface{}) error
	SessionID() string
}

//var globalSessions *session.Manager
var globalSessions *Manager
var provides = make(map[string]Provider)
var pder = &Providers{list: list.New()} //memory.goとの結合

func NewManager(providerName, cookieName string, maxlifetime int64) (*Manager, error) {
	provider, ok := provides[providerName]
	if !ok {
		return nil, fmt.Errorf("session: unknown provide %q (forgotten import?)", providerName)
	}
	return &Manager{provider: provider, cookieName: cookieName, maxlifetime: maxlifetime}, nil
}

func init() {
	pder.sessions = make(map[string]*list.Element, 0)
	Register("memory", pder)
	globalSessions, _ = NewManager("memory", "gosessionid", 3600)
	go globalSessions.GC()
}

func (manager *Manager) GC() {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	manager.provider.SessionGC(manager.maxlifetime)
	time.AfterFunc(time.Duration(manager.maxlifetime), func() {
		manager.GC()
	})
}

func Register(name string, provider Provider) {
	if provider == nil {
		panic("session: Register provider is nil")
	}
	if _, dup := provides[name]; dup {
		panic("session: Register called twice for provide" + name)
	}
	provides[name] = provider
}

//sessionIdがグローバルでユニークであることを保証
func (manager *Manager) sessionId() string {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

// memory.go
type Providers struct {
	lock     sync.Mutex
	sessions map[string]*list.Element
	list     *list.List
}

type SessionStore struct {
	sid          string
	timeAccessed time.Time
	value        map[interface{}]interface{}
}

func (st *SessionStore) Set(key, value interface{}) error {
	st.value[key] = value
	pder.SessionUpdate(st.sid)
	return nil
}

func (st *SessionStore) Get(key interface{}) interface{} {
	pder.SessionUpdate(st.sid)
	if v, ok := st.value[key]; ok {
		return v
	} else {
		return nil
	}
	return nil
}

func (st *SessionStore) Delete(key interface{}) error {
	delete(st.value, key)
	pder.SessionUpdate(st.sid)
	return nil
}

func (st *SessionStore) SessionID() string {
	return st.sid
}

//Sessionが現在アクセスしているユーザと既に関係しているか検査
func (manager *Manager) SessionStart(w http.ResponseWriter, r *http.Request) (session Session) {
	manager.lock.Lock()
	defer manager.lock.Unlock()
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == "" {
		sid := manager.sessionId()
		session, _ = manager.provider.SessionInit(sid)
		cookie := http.Cookie{Name: manager.cookieName, Value: url.QueryEscape(sid),
			Path: "/", HttpOnly: true, MaxAge: int(manager.maxlifetime)}
		http.SetCookie(w, &cookie)
	} else {
		sid, _ := url.QueryUnescape(cookie.Value)
		session, _ = manager.provider.SessionRead(sid)
	}
	return
}

func (manager *Manager) SessionDestroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(manager.cookieName)
	if err != nil || cookie.Value == "" {
		return
	} else {
		manager.lock.Lock()
		defer manager.lock.Unlock()
		manager.provider.SessionDestroy(cookie.Value)
		expiration := time.Now()
		cookie := http.Cookie{Name: manager.cookieName, Path: "/", HttpOnly: true,
			Expires: expiration, MaxAge: -1}
		http.SetCookie(w, &cookie)
	}
}

func (pder *Providers) SessionInit(sid string) (Session, error) {
	pder.lock.Lock()
	defer pder.lock.Unlock()
	v := make(map[interface{}]interface{}, 0)
	newsess := &SessionStore{sid: sid, timeAccessed: time.Now(), value: v}
	element := pder.list.PushBack(newsess)
	pder.sessions[sid] = element
	return newsess, nil
}

func (pder *Providers) SessionRead(sid string) (Session, error) {
	if element, ok := pder.sessions[sid]; ok {
		return element.Value.(*SessionStore), nil
	} else {
		sess, err := pder.SessionInit(sid)
		return sess, err
	}
	return nil, nil
}

func (pder *Providers) SessionDestroy(sid string) error {
	if element, ok := pder.sessions[sid]; ok {
		delete(pder.sessions, sid)
		pder.list.Remove(element)
		return nil
	}
	return nil
}

func (pder *Providers) SessionGC(maxlifetime int64) {
	pder.lock.Lock()
	defer pder.lock.Unlock()

	for {
		element := pder.list.Back()
		if element == nil {
			break
		}
		if (element.Value.(*SessionStore).timeAccessed.Unix() + maxlifetime) < time.Now().Unix() {
			pder.list.Remove(element)
			delete(pder.sessions, element.Value.(*SessionStore).sid)
		} else {
			break
		}
	}
}

func (pder *Providers) SessionUpdate(sid string) error {
	pder.lock.Lock()
	defer pder.lock.Unlock()
	if element, ok := pder.sessions[sid]; ok {
		element.Value.(*SessionStore).timeAccessed = time.Now()
		pder.list.MoveToFront(element)
		return nil
	}
	return nil
}

func DbConnect() (db *sql.DB) {
	db, err := sql.Open("mysql", "root:"+os.Getenv("DB_PASSWORD")+"@tcp(127.0.0.1:3306)/bbs")
	if err != nil {
		log.Println(err)
	}
	return db
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	//sess, _ := globalSessions.SessionStart(w, r)
	sess := globalSessions.SessionStart(w, r)
	switch r.Method {
	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("public/login.html"))
		w.Header().Set("Content-Type", "text/html")
		email := sess.Get("email")
		fmt.Println(email)
		tmpl.Execute(w, email)
	case http.MethodPost:
		var id string
		email := r.FormValue("email")
		pw := r.FormValue("password")

		fmt.Println("email:", r.Form["email"])
		sess.Set("email", r.Form["email"])

		db := DbConnect()
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

func SignupHandler(w http.ResponseWriter, r *http.Request) {
	db := DbConnect()
	defer db.Close()

	switch r.Method {
	case http.MethodGet:
		tmpl := template.Must(template.ParseFiles("public/signup.html"))
		tmpl.Execute(w, nil)
	case http.MethodPost:
		name := r.FormValue("name")
		email := r.FormValue("email")
		pw := r.FormValue("password")

		db := DbConnect()
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

func StaticHandler(w http.ResponseWriter, r *http.Request) {
	db := DbConnect()
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

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/signup", SignupHandler)
	http.HandleFunc("/index", StaticHandler)
	s := &http.Server{
		Addr: ":9000",
	}

	fmt.Println("server start port:8080")
	s.ListenAndServe()
}
