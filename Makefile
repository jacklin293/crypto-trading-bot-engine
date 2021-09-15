run:
	go build && ./crypto-trading-bot-main
test:
	go test -count=1 ./...
