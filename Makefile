
cluster:
	k3d cluster create stream-platform-cluster \
		--agents 1 \
		-p 8089:8089/TCP@agent:0 \
		-p 3478:3478/UDP@agent:0 \
		--registry-create k3d-stream-platform-registry:50000

delete:
	k3d cluster delete stream-platform-cluster

helm:
	helm install stream-platform ./infra/k8s

charts:
	helm dependencies build ./infra/k8s

build:
	REGISTRY=localhost:50000 docker-compose down --rmi all && \
			 REGISTRY=localhost:50000 docker-compose build && \
			 REGISTRY=localhost:50000 docker-compose push

deploy:
	helm dep update infra/k8s && \
		helm install stream-platform infra/k8s

