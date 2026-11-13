.PHONY: up down build diagrams codegen k8s-lint helm-lint actionlint test k8s-up k8s-down k8s-test istio-up istio-cb istio-test precommit

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

# istio-цели: trap-teardown не применяется по тем же причинам, что и для k8s-*.
# Порядок gateway -> istiod осознанный (так в задании); из-за него под gateway
# создаётся с image "auto" до поднятия webhook'а istiod, поэтому после
# установки istiod делаем rollout restart gateway — иначе ImagePullBackOff.
istio-up:
	minikube status >/dev/null 2>&1 || { echo "ERROR: minikube не запущен"; exit 1; }; \
		helm repo add istio https://istio-release.storage.googleapis.com/charts && \
		helm repo update && \
		helm upgrade --install istio-base istio/base -n istio-system --set defaultRevision=default --create-namespace && \
		helm upgrade --install istio-ingressgateway istio/gateway -n istio-system && \
		helm upgrade --install istiod istio/istiod -n istio-system --wait \
			--set pilot.resources.requests.cpu=100m \
			--set pilot.resources.requests.memory=512Mi && \
		kubectl rollout restart deployment istio-ingressgateway -n istio-system && \
		kubectl rollout status deployment istio-ingressgateway -n istio-system && \
		kubectl label namespace cinemaabyss istio-injection=enabled --overwrite && \
		kubectl patch statefulset postgres -n cinemaabyss -p '{"spec":{"template":{"metadata":{"annotations":{"sidecar.istio.io/inject":"false"}}}}}' && \
		kubectl patch statefulset kafka -n cinemaabyss -p '{"spec":{"template":{"metadata":{"annotations":{"sidecar.istio.io/inject":"false"}}}}}' && \
		kubectl patch statefulset zookeeper -n cinemaabyss -p '{"spec":{"template":{"metadata":{"annotations":{"sidecar.istio.io/inject":"false"}}}}}' && \
		kubectl rollout restart deployment,statefulset -n cinemaabyss && \
		kubectl apply -f src/kubernetes/istio-gateway.yaml && \
		kubectl get pods -n istio-system

istio-cb:
	kubectl apply -f src/kubernetes/circuit-breaker-config.yaml -n cinemaabyss && \
		kubectl get destinationrules -n cinemaabyss

# Fortio-тест circuit breaker'а: при -c 50 ожидается ~80% ответов 503
# (upstream_rq_pending_overflow > 0 в стате sidecar'а fortio).
istio-test:
	kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.25/samples/httpbin/sample-client/fortio-deploy.yaml -n cinemaabyss && \
		kubectl wait --for=condition=ready pod -l app=fortio -n cinemaabyss --timeout=120s && \
		FORTIO_POD=$$(kubectl get pod -n cinemaabyss -l app=fortio -o jsonpath='{.items[0].metadata.name}') && \
		kubectl exec -n cinemaabyss $$FORTIO_POD -c fortio -- fortio load -c 50 -qps 0 -n 500 -loglevel Warning http://movies-service:8081/api/movies && \
		kubectl exec -n cinemaabyss $$FORTIO_POD -c istio-proxy -- pilot-agent request GET stats | grep movies-service | grep pending
