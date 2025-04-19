CREATE DATABASE IF NOT EXISTS images;

USE images;

CREATE TABLE IF NOT EXISTS Images (
    id INT AUTO_INCREMENT PRIMARY KEY,
    Description TEXT,
    Personfound TEXT,
    Image LONGBLOB
);
