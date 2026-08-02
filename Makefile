.PHONY: help build run test clean

help:
	@echo "ISOTOPE - команды сборки"
	@echo ""
	@echo "  make build     - собрать ядро"
	@echo "  make run       - запустить узлы в Docker"
	@echo "  make test      - прогнать все тесты"
	@echo "  make clean     - очистить сборку"

build:
	cd node && go build -o sbimain .

run:
	docker compose build --no-cache
	docker compose up -d

test:
	cd node && go test -v ./...

clean:
	docker compose down
	rm -f node/sbimain
	rm -rf state/state_node*.json