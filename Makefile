.PHONY: up down lint test build-go

up:
	docker compose up --build

down:
	docker compose down -v

lint:
	cd backend-go && go run . lint ../rules

test:
	cd backend-go && go test ./...

build-go:
	cd backend-go && go build -o wraith .

mutation-test:
	python3 engine-python/mutation_testing.py --rule rules/suspicious_powershell_encodedcommand.yml --out /tmp/wraith-mutants.json

navigator:
	python3 engine-python/attack_navigator.py --rules rules/*.yml --out /tmp/wraith-attack-navigator.json

quality-test:
	python3 -m pytest tests/test_advanced_features.py -q
