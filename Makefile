all: dev

.PHONY: glide
glide:
	glide install

.PHONY: db
db:
	cd ./sys && \
	if [ -f "media.db" ]; then rm media.db; fi && \
	sqlite3 media.db < dump.sql

.PHONY: build
build: glide db
	CGO_ENABLED=1 go build -a -installsuffix cgo -o ./media-service.o .

.PHONY: recompile
recompile:
	CGO_ENABLED=1 go build -i -installsuffix cgo -o ./media-service.o .

.PHONY: docker
docker:
	docker build -t media-service:latest .
