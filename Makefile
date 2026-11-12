.PHONY: up down build diagrams codegen k8s-lint helm-lint actionlint test k8s-up k8s-down k8s-test precommit

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

k8s-lint:
	docker compose run --rm kubeconform

# pipefail обязателен: иначе падение helm template маскируется пустым
# stdin kubeconform (0 ресурсов = успех)
helm-lint: SHELL := /bin/bash
helm-lint:
	set -o pipefail; docker compose run --rm --no-deps helm template cinemaabyss src/kubernetes/helm --namespace cinemaabyss | docker compose run --rm --no-deps -T kubeconform -strict -summary -

actionlint:
	docker compose run --rm actionlint

test:
	trap 'docker compose down' EXIT; \
		docker compose up -d --build --wait postgres zookeeper kafka monolith movies-service events-service proxy-service && \
		echo "Waiting for services to be ready..." && \
		docker build -t cinemaabyss-api-tests tests/postman && \
		docker run --rm --network=cinemaabyss-network cinemaabyss-api-tests

# k8s-цели: trap-teardown к ним осознанно не применяется —
# кластер minikube не гасим, чтобы не ломать окружение разработчика.
# Манифесты применяются ЯВНЫМ списком в порядке зависимостей
# (namespace первым, kafka/ — отдельным -f): алфавитный apply недопустим.
k8s-up:
	minikube status >/dev/null 2>&1 || minikube start; \
		minikube addons enable ingress; \
		kubectl apply -f src/kubernetes/namespace.yaml && \
		kubectl apply -f src/kubernetes/configmap.yaml -f src/kubernetes/secret.yaml -f src/kubernetes/dockerconfigsecret.yaml -f src/kubernetes/postgres-init-configmap.yaml && \
		kubectl apply -f src/kubernetes/postgres.yaml && \
		kubectl apply -f src/kubernetes/kafka/ && \
		kubectl apply -f src/kubernetes/monolith.yaml && \
		kubectl apply -f src/kubernetes/movies-service.yaml && \
		kubectl apply -f src/kubernetes/events-service.yaml && \
		kubectl apply -f src/kubernetes/proxy-service.yaml && \
		kubectl apply -f src/kubernetes/ingress.yaml

k8s-down:
	kubectl delete -f src/kubernetes/ingress.yaml --ignore-not-found && \
		kubectl delete -f src/kubernetes/proxy-service.yaml --ignore-not-found && \
		kubectl delete -f src/kubernetes/events-service.yaml --ignore-not-found && \
		kubectl delete -f src/kubernetes/movies-service.yaml --ignore-not-found && \
		kubectl delete -f src/kubernetes/monolith.yaml --ignore-not-found && \
		kubectl delete -f src/kubernetes/kafka/ --ignore-not-found && \
		kubectl delete -f src/kubernetes/postgres.yaml --ignore-not-found && \
		kubectl delete -f src/kubernetes/configmap.yaml -f src/kubernetes/secret.yaml -f src/kubernetes/dockerconfigsecret.yaml -f src/kubernetes/postgres-init-configmap.yaml --ignore-not-found && \
		kubectl delete -f src/kubernetes/namespace.yaml --ignore-not-found

k8s-test:
	@echo "NOTE: requires hosts entry cinemaabyss.example.com -> minikube ip and a running 'minikube tunnel'"
	@curl -sf --max-time 5 http://cinemaabyss.example.com/health >/dev/null 2>&1 || { \
		echo "ERROR: http://cinemaabyss.example.com недоступен — тесты запускать бессмысленно."; \
		echo "Проверьте:"; \
		echo "  1) в /etc/hosts есть запись: 127.0.0.1 cinemaabyss.example.com"; \
		echo "  2) 'minikube tunnel' запущен ПОД sudo (порты 80/443 привилегированные;"; \
		echo "     в логе tunnel не должно быть 'sudo: timed out');"; \
		echo "  3) minikube и ingress-addon работают: minikube status, kubectl -n cinemaabyss get pods"; \
		exit 1; \
	}
	npm --prefix tests/postman run test:kubernetes

precommit:
	$(MAKE) diagrams
	$(MAKE) codegen
	$(MAKE) build
	$(MAKE) k8s-lint
	$(MAKE) helm-lint
	$(MAKE) actionlint
	$(MAKE) test
