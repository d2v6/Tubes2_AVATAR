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
COPY go.mod go.sum ./
RUN go mod download
COPY ./src/backend ./src/backend
COPY --from=frontend /app/frontend/dist ./src/backend/frontend/dist
RUN go build -o server ./src/backend/main.go

# Final image
FROM debian:stable-slim
WORKDIR /app
COPY --from=backend /app/server ./
COPY --from=backend /app/src/backend/frontend/dist ./frontend/dist
EXPOSE 8080
CMD ["./server"]