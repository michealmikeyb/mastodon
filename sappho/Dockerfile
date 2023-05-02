
FROM golang:buster



# Move to working directory /app
WORKDIR /app

COPY ./go.mod ./
COPY ./go.sum ./

# Download dependencies using go mod
RUN go mod download

# Copy the code into the container
COPY . .




# Build the application's binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o server ./cmd/server.go

RUN chmod 777 /app/server


# Command to run the application when starting the container
CMD ["/app/server"]