DOCKER_ARGS=
APP=ap-5r
PROJECT=swgoh-api
VERSION=latest
TOKEN=$(shell cat .token 2>/dev/null)

build:
	rm -f ap-5r && GOARCH=amd64 GOOS=linux go build -o $(APP)
	docker build \
		-t ronoaldo/$(APP):$(VERSION) \
		--build-arg GIT_HASH=$$(git rev-parse --short HEAD) .

run: build
	(docker ps | grep pagerender) || docker run -d --rm --name pagerender ronoaldo/pagerender
	docker run --name ap-5r \
		--rm \
		--link pagerender \
		--env USE_DEV=true \
		--env BOT_TOKEN=$(TOKEN) \
		--env API_USERNAME=$(SWGOH_API_USERNAME) \
		--env API_PASSWORD=$(SWGOH_API_PASSWORD) \
		--env SWGOH_CACHE_DIR=/tmp/cache \
		--volume $(HOME)/.cache/api.swgoh.help:/tmp/cache \
		-it $(DOCKER_ARGS) \
		ronoaldo/$(APP):$(VERSION)

deploy: build
	docker push ronoaldo/$(APP):$(VERSION)
