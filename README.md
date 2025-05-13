# Tubes2_AVATAR

## Description
Sebuah aplikasi yang mengimplementasikan algoritma DFS dan BFS untuk menemukan resep kombinasi elemen, mirip seperti pada permainan "Little Alchemy". Aplikasi ini menemukan berbagai cara untuk menciptakan elemen dengan menggabungkan elemen-elemen dasar melalui beberapa tahapan.

## Algorithm Implementation
1. Breadth-First Search (BFS)
    - Dimulai dari elemen target, algoritma mengidentifikasi semua kombinasi elemen induk yang dibutuhkan.
    - Elemen diproses secara bertingkat, dimulai dari elemen (level) yang paling dekat dengan target.
    - Elemen diproses menggunakan pendekatan berbasis antrean (queue), memastikan semua elemen pada jarak yang sama dari target diproses sebelum berpindah ke tingkat berikutnya.
    - Melacak elemen yang telah diproses untuk menghindari perhitungan yang berulang.
    - Pohon resep dibangun dari bawah ke atas (bottom-up), dengan menggabungkan pohon elemen induk untuk membentuk pohon elemen anak.

2. Depth-First Search (DFS)
    - Dimulai dari elemen target, algoritma akan secara rekursif menelusuri setiap kemungkinan resep (kombinasi elemen induk) yang dapat membentuk elemen tersebut.
    - Untuk setiap resep, algoritma akan secara rekursif menelusuri cara membuat masing-masing elemen induknya. 
    - Penelusuran dilanjutkan hingga mencapai elemen-elemen dasar (elemen dengan tier 0 atau elemen tanpa induk).
    - Algoritma akan membangun pohon resep, di mana setiap node merepresentasikan suatu elemen dan anak-anaknya adalah elemen yang dibutuhkan untuk membuatnya.

## Requirements
- Docker Desktop (to run using Docker)

## How to Run (Terminal Based)
1. Clone the Repository
    ```
    git clone https://github.com/d2v6/Tubes2_AVATAR.git
    ```
2. Install Dependencies and Run frontend Directory
    ```
    cd src/frontend
    npm install
    npm run dev
    ```
3. Run backend Directory
    ```
    cd src/backend
    go run main.go
    ```

## How to Run (Docker Based)
1. Clone the Repository
    ```
    git clone https://github.com/d2v6/Tubes2_AVATAR.git
    ```
2. Make Sure Docker Desktop is Installed
3. Build Docker Image
    ```
    docker build -t avatar-tubes2 .
    ```
4. Run Docker Container
    ```
    docker run -p 4003:4003 avatar-tubes2
    ```

## Author
| NIM  | Nama |
|------|------|
| 13523003 | Dave Daniell Yanni |
| 13523029 | Bryan Ho |
| 13523099 | Daniel Pedrosa Wu |


