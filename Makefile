.PHONY: test test-integration test-integration-local docker-up docker-down build install release clean help

help:
	@echo "Available targets:"
	@echo "  test                   - Run unit tests"
	@echo "  test-integration       - Run integration tests (requires Neo4j)"
	@echo "  test-integration-local - Start Neo4j with Docker and run integration tests"
	@echo "  docker-up              - Start Neo4j with Docker Compose"
	@echo "  docker-down            - Stop Neo4j Docker Compose"
	@echo "  build                  - Build CLI binary"
	@echo "  install                - Install CLI to \$$GOPATH/bin"
	@echo "  release                - Create release (goreleaser)"
	@echo "  clean                  - Clean build artifacts"

test:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

test-integration:
	NEO4J_URI=bolt://localhost:7687 \
	NEO4J_USERNAME=neo4j \
	NEO4J_PASSWORD=testpassword \
	NEO4J_DATABASE=neo4j \
	go test -v -race -tags=integration -coverprofile=coverage-integration.out -covermode=atomic ./...

test-integration-local: docker-up
	@echo "Waiting for Neo4j to be ready..."
	@sleep 10
	@$(MAKE) test-integration
	@$(MAKE) docker-down

docker-up:
	docker-compose up -d
	@echo "Waiting for Neo4j to be ready..."
	@sleep 15

docker-down:
	docker-compose down -v

build:
	go build -o bin/neo4go ./cmd/neo4go

install:
	go install ./cmd/neo4go

release:
	goreleaser release --clean

clean:
	rm -rf bin/ dist/ coverage.out
