.PHONY: build run clean

build:
	go build -o bin/sqpix.exe cmd/mcp/main.go
	
run:
	go run cmd/mcp/main.go -server 10.110.104.4 -user sa -password P@ssw0rd -database DSV_PIX

clean:
	if exist bin rmdir /s /q bin
