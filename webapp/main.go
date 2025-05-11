package main

import (
	"context"
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
	amqp "github.com/rabbitmq/amqp091-go"
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

type QueueMessage struct {
	PersonFound int    `json:"personFound"`
	Description string `json:"description"`
}

var EXCHANGE_NAME = "IMAGE_MESSAGESV2"

func getImages(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT Description, Personfound, Image FROM Images")
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
	var message = QueueMessage{
		PersonFound: imageData.PersonFound,
		Description: description,
	}
	sendMessageToQueue(message)
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
func sendMessageToQueue(message QueueMessage) {
	rabbitmq := os.Getenv("RABBITMQ_ADDR")
	conn, err := amqp.Dial("amqp://guest:guest@" + rabbitmq + ":5672/")
	if err != nil {
		log.Println("Error during connecting to rabbitMQ: ", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Println("Error during creating a channel: ", err)
	}

	err = ch.ExchangeDeclare(
		EXCHANGE_NAME, // name
		"fanout",      // type
		true,          // durable
		false,         // auto-deleted
		false,         // internal
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Println("There was an error during decalring exchange: ", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	codedMessage, err := json.Marshal(message)
	if err != nil {
		log.Println("There was an error with the marshaling of the message: ", err)
	}
	err = ch.PublishWithContext(ctx,
		EXCHANGE_NAME, // exchange
		"",            // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         codedMessage,
		})
	log.Printf("message sent: %+v", message)
}
func messagesStream(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	rabbitmq := os.Getenv("RABBITMQ_ADDR")
	conn, err := amqp.Dial("amqp://guest:guest@" + rabbitmq + ":5672/")
	if err != nil {
		log.Println("RabbitMQ connection error:", err)
		http.Error(w, "Failed to connect to RabbitMQ", http.StatusInternalServerError)
		return
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Println("RabbitMQ channel error:", err)
		http.Error(w, "Failed to create channel", http.StatusInternalServerError)
		return
	}

	err = ch.ExchangeDeclare(EXCHANGE_NAME, "fanout", true, false, false, false, nil)
	if err != nil {
		log.Println("Exchange declare error:", err)
	}

	q, err := ch.QueueDeclare(username, true, false, false, false, nil)
	if err != nil {
		log.Println("Queue declare error:", err)
	}

	err = ch.QueueBind(q.Name, "", EXCHANGE_NAME, false, nil)
	if err != nil {
		log.Println("Queue bind error:", err)
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Println("Queue consume error:", err)
		http.Error(w, "Failed to consume", http.StatusInternalServerError)
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()

	log.Println("Started streaming for user:", username)

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				log.Println("RabbitMQ channel closed")
				return
			}
			_, err := fmt.Fprintf(w, "data: %s\n\n", msg.Body)
			if err != nil {
				log.Println("Write error:", err)
				return
			}
			flusher.Flush()
		case <-ctx.Done():
			log.Println("Client disconnected:", username)
			ch.Close()
			conn.Close()
			return
		}
	}
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
	if os.Getenv("MODE") == "Production" {
		content, err := fs.Sub(staticFiles, "static")
		if err != nil {
			log.Println("There was a problem with the strip of the static directory: ", err)
		}
		fs := http.FS(content)
		http.Handle("/", http.FileServer(fs))
	} else {
		http.Handle("/", http.FileServer(http.Dir("./static")))
	}

	http.HandleFunc("/addImage", addImage)
	http.HandleFunc("/getImages", getImages)
	http.HandleFunc("/messages", messagesStream)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server error:", err)
	}

}
