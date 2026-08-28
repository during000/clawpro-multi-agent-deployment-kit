APP_NAME := hatchery
BUILD_DIR := build

.PHONY: build release clean run test openapi

build:
	go build -race -o $(APP_NAME) .

release:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -tags release -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"
	@ls -lh $(BUILD_DIR)/$(APP_NAME)

clean:
	rm -rf $(BUILD_DIR) $(APP_NAME)

run: build
	./$(APP_NAME) --region ap-guangzhou --init-user admin --init-pass 123456

# Generate OpenAPI spec from API.md (required before integration tests)
openapi:
	python3 test/api_md_to_openapi.py
ifdef BASE_BRANCH
	@echo "Generating base branch OpenAPI spec for incremental coverage..."
	@git show $(BASE_BRANCH):docs/API.md > /tmp/base_api.md 2>/dev/null && \
		python3 test/api_md_to_openapi.py --input /tmp/base_api.md --output docs/openapi_base.json && \
		rm -f /tmp/base_api.md || \
		echo "Warning: failed to generate base spec from $(BASE_BRANCH), skipping incremental"
endif

# Integration test: build and run the K8s-based integration test runner.
# Deploys hatchery to K8s, runs init.py + test scripts, then cleans up.
#
# Usage:
#   make test IMAGE=your-registry/hatchery:tag
#   make test IMAGE=your-registry/hatchery:tag TEST_ARGS="--no-cleanup"
#
# Variables:
#   IMAGE      - Container image to deploy (required)
#   TEST_ARGS  - Extra flags passed to the test runner (optional)
#                e.g. --no-cleanup, --concurrency 5, --script-timeout 20m
TEST_DIR := test/integration
TEST_BIN := $(BUILD_DIR)/integration-test

test: openapi
ifndef IMAGE
	$(error IMAGE is required. Usage: make test IMAGE=your-registry/hatchery:tag)
endif
	@mkdir -p $(BUILD_DIR)
	cd $(TEST_DIR) && go build -o ../../$(TEST_BIN) .
	./$(TEST_BIN) --image $(IMAGE) $(TEST_ARGS)
