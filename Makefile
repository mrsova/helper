.PHONY: build-darwin
build-darwin: ## Build for darwin
	GOOS=darwin GOARCH=amd64 go build -o build/app-mac main.go

.PHONY: build-linux
build-linux: ## Build for linux
	GOOS=linux GOARCH=amd64 go build -o build/app-linux main.go

.PHONY: build-windows
build-windows: ## Build for windows
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o build/app-windows.exe main.go
