.PHONY: test
test:
	go test ./... -v

.PHONY: coverage
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: coverage-check
coverage-check:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | findstr "total:"
	@echo "Target: >80% coverage"