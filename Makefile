.PHONY: up down build diagrams test precommit

up:
	docker compose up -d

down:
	docker compose down -v

build:
	docker compose build

diagrams:
	docker compose run --rm --build --user "$$(id -u):$$(id -g)" plantuml

test:
	docker compose up -d --build postgres zookeeper kafka monolith movies-service events-service proxy-service
	@echo "Waiting for services to be ready..."; sleep 30
	docker build -t cinemaabyss-api-tests tests/postman
	docker run --rm --network=cinemaabyss-network cinemaabyss-api-tests; rc=$$?; docker compose down; exit $$rc

precommit:
	$(MAKE) diagrams
	$(MAKE) build
	$(MAKE) test
