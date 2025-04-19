from flask import Flask, request, jsonify
from ultralytics import YOLO
import cv2
import numpy as np
import base64
import io

app = Flask(__name__)
model = YOLO("yolov8n.pt")  # or yolov8s.pt for better accuracy

@app.route('/detectHuman', methods=['POST'])
def detect_human():
    try:
        # Decode image from request
        image_bytes = request.data
        nparr = np.frombuffer(image_bytes, np.uint8)
        img = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

        # YOLOv8 detection
        results = model(img)
        person_count = 0

        # Draw only 'person' boxes
        for result in results:
            for box in result.boxes:
                cls = int(box.cls[0])
                if model.names[cls] == 'person':
                    person_count += 1
                    x1, y1, x2, y2 = map(int, box.xyxy[0])
                    cv2.rectangle(img, (x1, y1), (x2, y2), (0, 255, 0), 2)

        # Convert image to base64
        _, buffer = cv2.imencode('.jpg', img)
        image_base64 = base64.b64encode(buffer).decode('utf-8')

        return jsonify({
            'image': image_base64,
            'personFound': person_count
        })

    except Exception as e:
        return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    app.run(debug=True, host='0.0.0.0', port=5000)
