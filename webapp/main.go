package main

import (
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// Struct to return to the client
type ImageResponse struct {
	Description string `json:"description"`
	ImageData   string `json:"imageData"`
	PersonFound int    `json:"personFound"`
	ContentType string `json:"contentType"` // optional: we'll just use image/jpeg for now
}

type DetectionServiceResponse struct {
	PersonFound   int    `json:"personFound"`
	DetectedImage string `json:"image"`
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
		var personfound int
		var imgBytes []byte

		if err := rows.Scan(&desc, &personfound, &imgBytes); err != nil {
			log.Println("There was an error during the scanning:", err)
			continue
		}

		encoded := base64.StdEncoding.EncodeToString(imgBytes)

		res = append(res, ImageResponse{
			Description: desc,
			PersonFound: personfound,
			ImageData:   encoded,
			ContentType: "image/jpeg", // or detect type if needed
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func addImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		log.Println("Wrong HTTP method for posting image")
		return
	}

	err := r.ParseMultipartForm(10 << 20) // 10MB max memory
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

	client := http.Client{Timeout: 5 * time.Second}
	objectDetectServiceAddr := "http://" + os.Getenv("HUMAN_DETECTION_SERVICE_ADDR") + ":" + os.Getenv("HUMAN_DETECTION_SERVICE_PORT") + "/detectHuman"

	// Send the image directly as a POST body
	resp, err := client.Post(objectDetectServiceAddr, "image/jpeg", file)
	if err != nil {
		http.Error(w, "Failed to call detection service", http.StatusInternalServerError)
		log.Println("Error sending POST request to detection service:", err)
		return
	}
	defer resp.Body.Close()

	// Decode the JSON into your struct
	var imageData DetectionServiceResponse
	err = json.NewDecoder(resp.Body).Decode(&imageData)
	if err != nil {
		http.Error(w, "Failed to decode response JSON", http.StatusInternalServerError)
		log.Println("Error decoding JSON:", err)
		return
	}

	// Decode base64 image
	decodedImageBytes, err := base64.StdEncoding.DecodeString(imageData.DetectedImage)
	if err != nil {
		http.Error(w, "Failed to decode image", http.StatusInternalServerError)
		log.Println("Failed to decode base64 image:", err)
		return
	}

	// Save to DB
	_, err = db.Exec("INSERT INTO Images (Description, Personfound, Image) VALUES (?, ?, ?)", description, imageData.PersonFound, decodedImageBytes)
	if err != nil {
		http.Error(w, "Database insert error", http.StatusInternalServerError)
		log.Println("Error inserting image:", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("Image uploaded successfully. Persons detected: %d", imageData.PersonFound)))
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

//go:embed static/index.html
var staticFiles embed.FS

func main() {
	db = initDB()
	pingErr := db.Ping()

	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")
	content, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Println("There was a problem with the strip of the static directory: ", err)
	}
	fs := http.FS(content)
	http.Handle("/", http.FileServer(fs))
	http.HandleFunc("/addImage", addImage)
	http.HandleFunc("/getImages", getImages)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server error:", err)
	}

}
