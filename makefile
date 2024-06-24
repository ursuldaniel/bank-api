build: 
	@go build -o ./bin/bank-api ./cmd/main/main.go

run: build
	@./bin/bank-api