package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"encoding/base64"
	"encoding/json"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// Struct to return to the client
type ImageResponse struct {
	Description string `json:"description"`
	ImageData   string `json:"imageData"`
	ContentType string `json:"contentType"` // optional: we'll just use image/jpeg for now
}

func getImages(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT Description, Image FROM Images")
	if err != nil {
		http.Error(w, "Error querying database", http.StatusInternalServerError)
		log.Println("Error getting the rows:", err)
		return
	}
	defer rows.Close()

	var res []ImageResponse

	for rows.Next() {
		var desc string
		var imgBytes []byte

		if err := rows.Scan(&desc, &imgBytes); err != nil {
			log.Println("There was an error during the scanning:", err)
			continue
		}

		encoded := base64.StdEncoding.EncodeToString(imgBytes)

		res = append(res, ImageResponse{
			Description: desc,
			ImageData:   encoded,
			ContentType: "image/jpeg", // or detect type if needed
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func addImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("Wrong HTTP metho for posting image")
		return
	}
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		log.Println("Error parsing form:", err)
		return
	}
	description := r.FormValue("description")

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Failed to get image file", http.StatusBadRequest)
		log.Println("Error getting file:", err)
		return
	}
	defer file.Close()
	imageBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		log.Println("Error reading image:", err)
		return
	}
	_, err = db.Exec("INSERT INTO Images (Description, Image) VALUES (?, ?)", description, imageBytes)
	if err != nil {
		http.Error(w, "Database insert error", http.StatusInternalServerError)
		log.Println("Error inserting image:", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Image uploaded successfully"))

}

func initDB() *sql.DB {
	if err := godotenv.Load(); err != nil {
		log.Println("Error during reading env variables: ", err)
	}

	cfg := mysql.NewConfig()
	cfg.User = "root"
	cfg.Passwd = os.Getenv("MYSQL_PASSW")
	cfg.Net = "tcp"
	cfg.Addr = os.Getenv("DATABASE_ADDR") + ":" + os.Getenv("DATABASE_PORT")
	cfg.DBName = "images"
	var err error
	var db *sql.DB
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}
	return db

}

var db *sql.DB

func main() {
	db = initDB()
	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.HandleFunc("/addImage", addImage)
	http.HandleFunc("/getImages", getImages)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server error:", err)
	}

}
