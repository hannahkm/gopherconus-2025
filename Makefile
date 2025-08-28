fmt:
	golangci-lint fmt
	shfmt -w .

lint:
	golangci-lint run --fix

build:
	go build -o gopherconus-server main.go

demo:
	./loadtest.sh start
	./loadtest.sh run all

demo-stop:
	./loadtest.sh stop
