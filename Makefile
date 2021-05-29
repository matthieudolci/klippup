build:
	env GOOS=linux GOARCH=arm go build -o klippup main.go 
.PHONY: build