.PHONY: build

clean:
	rm -rf discord-photo-reaper

build:
	go mod tidy
	go build -o discord-photo-reaper .

test:
	go test -v .../.

run: build
	chmod +x discord-photo-reaper
	bash run.sh

docker: build
	bash docker-build.sh