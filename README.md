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

### Cloud / Deployment
- Each component runs as its own service.
- Docker is used to containerize and run them together.
- Demonstrates inter-service communication in a cloud-like environment.
