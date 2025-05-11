# Build frontend
FROM node:24-slim AS frontend
WORKDIR /app/frontend
COPY ./src/frontend/package.json ./src/frontend/package-lock.json ./
RUN npm install
COPY ./src/frontend ./
RUN npm run build

# Build Go backend
FROM golang:1.23 AS backend
WORKDIR /app
COPY src/backend/go.mod src/backend/go.sum ./
RUN go mod download
RUN apt-get update && apt-get install -y ca-certificates
RUN update-ca-certificates
COPY src/backend/ ./
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN go build -o server main.go

# Final image
FROM debian:stable-slim
WORKDIR /app
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=backend /app/server ./
COPY --from=backend /app/frontend/dist ./frontend/dist
RUN mkdir -p /app/src/backend/data
EXPOSE 4003
CMD ["./server"]
