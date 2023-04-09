package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"io/ioutil"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
)

var store = sessions.NewCookieStore([]byte("mysession"))

var db *sql.DB

var err error

func CreateAconut(res http.ResponseWriter, req *http.Request) {
	db := dbConn()
	if req.Method != "POST" {
		http.ServeFile(res, req, "static/templates/signUP.html")
		return

	}

	username := req.FormValue("Name")
	email := req.FormValue("email")
	password := req.FormValue("Password")

	var user string
	err := db.QueryRow("SELECT Name FROM productdb.Products WHERE Name=?", username).Scan(&user)
	switch {
	case err == sql.ErrNoRows:
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(res, "Server error, unable to create your account.", 500)
			return
		}

		_, err = db.Exec("insert into productdb.Products (Name, Email, Password) values(?,?,?)", username, email, hashedPassword)

		if err != nil {
			http.Error(res, "Server error, unable to create your account.", 500)
			return
		}
		res.Write([]byte("User created!"))

		return

	case err != nil:
		http.Error(res, "Server error, unable to create your account.", 500)
		return
	default:
		http.Redirect(res, req, "/", 301)
	}
	defer db.Close()
}

func loginPage(res http.ResponseWriter, req *http.Request) {
	db := dbConn()
	if req.Method != "POST" {
		http.ServeFile(res, req, "static/templates/signIn.html")
		return
	}

	username := req.FormValue("username")
	password := req.FormValue("password")

	var databaseUsername string
	var databasePassword string

	err := db.QueryRow("SELECT Name, Password FROM  productdb.Products  WHERE Name=?", username).Scan(&databaseUsername, &databasePassword)

	if err != nil {
		http.Redirect(res, req, "/login", 301)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(databasePassword), []byte(password))
	if err != nil {
		http.Redirect(res, req, "/login", 301)
		return
	}

	res.Write([]byte("Hello  " + databaseUsername))
	defer db.Close()
}

func Logout(response http.ResponseWriter, request *http.Request) {
	session, _ := store.Get(request, "mysession")
	session.Options.MaxAge = -1
	session.Save(request, response)
	http.Redirect(response, request, "/loginIndex", http.StatusSeeOther)
}

func index(w http.ResponseWriter, r *http.Request) {

	t, err := template.ParseFiles("static/templates/index.html")
	if err != nil {
		fmt.Fprintf(w, err.Error())
	}

	t.Execute(w, "index")
}
func dbConn() (db *sql.DB) {

	db, err = sql.Open("mysql", "root:root@/productdb")
	if err != nil {
		panic(err.Error())
	}
	return db
}

type upfile struct {
	ID        int
	Fname     string
	Item_type string
	Star      int
	Path      string
	Price     int
	Count     int
}

var tmpl = template.Must(template.ParseGlob("static/templates/*"))

func upload(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	var selDB *sql.Rows

	if r.Method == "POST" {
		rating := r.FormValue("raiting")
		price := r.FormValue("price")

		if rating == "raiting" {
			sel, err := db.Query("SELECT * FROM `upload` ORDER BY star ASC")
			selDB = sel
			if err != nil {
				panic(err.Error())
			}
		} else if price == "price" {
			sel, err := db.Query("SELECT * FROM `upload` ORDER BY price ASC")
			selDB = sel
			if err != nil {
				panic(err.Error())
			}
		} else {
			sel, err := db.Query("SELECT * FROM upload ORDER BY id DESC")
			selDB = sel
			if err != nil {
				panic(err.Error())
			}
		}
	} else {
		sel, err := db.Query("SELECT * FROM upload ORDER BY id DESC")
		selDB = sel
		if err != nil {
			panic(err.Error())
		}
	}

	upld := upfile{}
	res := []upfile{}
	for selDB.Next() {
		var id, star, price int
		var fname, item_type, path string

		err = selDB.Scan(&id, &fname, &item_type, &star, &path, &price)
		if err != nil {
			panic(err.Error())
		}
		upld.ID = id
		upld.Fname = fname
		upld.Item_type = item_type
		upld.Star = star
		upld.Path = path
		upld.Price = price
		res = append(res, upld)

	}

	upld.Count = len(res)

	if upld.Count > 0 {
		tmpl.ExecuteTemplate(w, "uploadfile.html", res)
	} else {
		tmpl.ExecuteTemplate(w, "uploadfile.html", nil)
	}

	db.Close()

}
func uploadFiles(w http.ResponseWriter, r *http.Request) {

	db := dbConn()
	if r.Method != "POST" {
		http.ServeFile(w, r, "static/templates/uploadfile.html")
		return
	}
	fname := r.FormValue("fname")
	item_type := r.FormValue("item_type")
	star := r.FormValue("star")
	price := r.FormValue("price")

	r.ParseMultipartForm(200000)
	if r == nil {
		fmt.Fprintf(w, "No files can be selected\n")
	}

	formdata := r.MultipartForm
	fil := formdata.File["files"]
	for i := range fil {
		file, err := fil[i].Open()
		if err != nil {
			fmt.Fprintln(w, err)
			return
		}
		defer file.Close()

		tempFile, err := ioutil.TempFile("static/assets/uploadimage/", "upload-*.jpg")

		if err != nil {
			fmt.Println(err)
		}
		defer tempFile.Close()

		filepath := tempFile.Name()
		fileBytes, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Println(err)
		}

		tempFile.Write(fileBytes)

		insForm, err := db.Prepare("INSERT INTO upload(fname, item_type, star, path, price) VALUES(?,?,?,?, ?)")
		if err != nil {
			panic(err.Error())
		} else {
			log.Println("data insert successfully . . .")
		}
		insForm.Exec(fname, item_type, star, filepath, price)

		log.Printf("Successfully Uploaded File\n")
		defer db.Close()

		http.Redirect(w, r, "/", 301)
	}

}
func delete(w http.ResponseWriter, r *http.Request) {
	db := dbConn()
	emp := r.URL.Query().Get("id")
	delForm, err := db.Prepare("DELETE FROM upload WHERE id=?")
	if err != nil {
		panic(err.Error())
	}
	delForm.Exec(emp)
	log.Println("deleted successfully")
	defer db.Close()
	http.Redirect(w, r, "/", 301)
}

func handleRequest() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.HandleFunc("/login", loginPage)
	http.HandleFunc("/signup", CreateAconut)
	http.HandleFunc("/logout", Logout)
	// http.HandleFunc("/", index)
	http.HandleFunc("/", upload)
	http.HandleFunc("/uploadfiles", uploadFiles)
	http.HandleFunc("/dele", delete)

	log.Println("Server started on: http://localhost:9000")

	http.ListenAndServe(":9000", nil)
}

func main() {

	handleRequest()

}
