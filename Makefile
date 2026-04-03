APP_NAME=go-cantika
DOCKER_USERNAME=gushim
VERSION=latest

IMAGE=$(DOCKER_USERNAME)/$(APP_NAME):$(VERSION)

build:
	docker buildx build \
  --platform linux/amd64 \
  -t $(IMAGE) \
  --push \
  .

push:
	docker push $(IMAGE)