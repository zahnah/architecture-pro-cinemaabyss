.PHONY: up down build diagrams codegen test precommit

up:
	docker compose up -d

down:
	docker compose down -v

build:
	docker compose build

diagrams:
	docker compose run --rm --build --user "$$(id -u):$$(id -g)" plantuml

codegen:
	docker compose run --rm --build --user "$$(id -u):$$(id -g)" codegen

test:
	trap 'docker compose down' EXIT; \
		docker compose up -d --build --wait postgres zookeeper kafka monolith movies-service events-service proxy-service && \
		echo "Waiting for services to be ready..." && \
		docker build -t cinemaabyss-api-tests tests/postman && \
		docker run --rm --network=cinemaabyss-network cinemaabyss-api-tests

precommit:
	$(MAKE) diagrams
	$(MAKE) codegen
	$(MAKE) build
	$(MAKE) test
