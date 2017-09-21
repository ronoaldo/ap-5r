build:
	go build -o discordbot
	docker build -t gcr.io/ronoaldoconsulting/discordbot:latest .

run: build
	docker run --rm -it gcr.io/ronoaldoconsulting/discordbot:latest

deploy: build
	gcloud docker -- push gcr.io/ronoaldoconsulting/discordbot:latest
