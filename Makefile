BINARY_AMD64 = proxytool-linux-amd64
BINARY_ARM64 = proxytool-linux-arm64
LDFLAGS = -ldflags="-s -w"

.PHONY: all linux-amd64 linux-arm64 clean

all: linux-amd64 linux-arm64

linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_AMD64) .
	@echo "Built: $(BINARY_AMD64)"

linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_ARM64) .
	@echo "Built: $(BINARY_ARM64)"

clean:
	rm -f $(BINARY_AMD64) $(BINARY_ARM64)
