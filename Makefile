build:
	go build -o discordbot
	docker build -t gcr.io/ronoaldoconsulting/discordbot:latest --build-arg GIT_HASH=$$(git rev-parse --short HEAD) .

run: build
	docker run --rm --env USE_DEV=true -it gcr.io/ronoaldoconsulting/discordbot:latest

deploy: build
	gcloud docker -- push gcr.io/ronoaldoconsulting/discordbot:latest
