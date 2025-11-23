## Overview

This project is a small cloud-based application built from three cooperating components: a **web frontend**, a **Python object-detection service**, and a **Go backend API**. Together, they demonstrate how multiple services can communicate in a cloud environment.

### 1. Web Application (Frontend)
- Allows the user to upload an image.
- Displays the processed image with detected humans.
- Built with simple HTML/CSS/JavaScript and sends the image to the Python service via HTTP.

### 2. Object Detection Service (Python)
- Receives the uploaded image from the web frontend.
- Uses a pretrained model to detect humans in the picture.
- Returns the image with bounding boxes drawn around detected people.
- Implemented in Python using common image-processing libraries.

### 3. Go Backend Service
- A lightweight API used to support the application and demonstrate running multiple microservices.
- Handles small tasks such as forwarding data or hosting simple logic (e.g., Tic Tac Toe logic from the lab task).
- Written in Go using the standard `net/http` library.

### 4. CI/CD Workflow

This project includes a GitHub Actions workflow that automates testing, building, and deploying the application. The workflow runs on every push and consists of three stages:

1. **Unit Testing**  
   - Checks out the code and sets up Go.  
   - Runs all Go tests inside the `webapp` directory.

2. **Build Stage**  
   - Builds the Go application into an executable (`detection_program`).  
   - Uploads the compiled binary as an artifact.

3. **Docker Build & Push**  
   - Downloads the built executable.  
   - Logs in to Docker Hub using repository secrets.  
   - Builds a Docker image using the executable and required build arguments.  
   - Pushes the image to the user's Docker Hub repository.

### Cloud / Deployment
- Each component runs as its own service.
- Docker is used to containerize and run them together.
- Demonstrates inter-service communication in a cloud-like environment.
