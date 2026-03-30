# Stage 1: Build the React UI
FROM node:24-alpine AS ui-builder
WORKDIR /app/ui
COPY ui/package*.json ./
RUN npm ci
COPY ui/ ./
RUN npm run build

# Stage 2: Build the Go binary
FROM golang:1.22-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/ui/dist ./ui/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o mcp-gateway .

# Stage 3: Minimal runtime image
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=go-builder /app/mcp-gateway .
EXPOSE 8080
ENTRYPOINT ["/app/mcp-gateway"]
