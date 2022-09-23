swagger-install:
	brew tap go-swagger/go-swagger
	brew install go-swagger
	which swagger || (GO111MODULE=off go get -u github.com/go-swagger/go-swagger/cmd/swagger)

swagger:
	swagger generate spec -o ./swagger.yaml --scan-models

serve-swagger:
	swagger serve -F=swagger swagger.yaml

swagger-run:
	which swagger || (GO111MODULE=off go get -u github.com/go-swagger/go-swagger/cmd/swagger)
	swagger generate spec -o ./swagger.yaml --scan-models
	swagger serve -F=swagger swagger.yaml

build-base:
	docker build -t neuron-base -f Dockerfile.base --rm  .

build-hack:
	docker build --tag=neuron -f Dockerfile.hack --rm . \
		&& kubectl rollout restart deployment/neuron -n phoenix
