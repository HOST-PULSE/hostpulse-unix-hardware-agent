# Имя готового бинарника
BINARY_NAME=unix-hardware-agent

# Флаги компиляции: -s -w убирают отладочную информацию для уменьшения размера файла
LDFLAGS=-ldflags="-s -w"

.PHONY: all build-linux build-freebsd build-all clean

# Команда по умолчанию (если просто написать 'make')
all: build-all

# Сборка только для Linux (архитектура x86_64)
build-linux:
	@echo " [🔨 BUILD] Компиляция агента под Linux amd64..."
	@mkdir -p bin/linux_amd64
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/linux_amd64/$(BINARY_NAME) main.go
	@echo " [✅ DONE] Файл собран: bin/linux_amd64/$(BINARY_NAME)"

# Сборка только для FreeBSD (архитектура x86_64)
build-freebsd:
	@echo " [🔨 BUILD] Компиляция агента под FreeBSD amd64..."
	@mkdir -p bin/freebsd_amd64
	GOOS=freebsd GOARCH=amd64 go build $(LDFLAGS) -o bin/freebsd_amd64/$(BINARY_NAME) main.go
	@echo " [✅ DONE] Файл собран: bin/freebsd_amd64/$(BINARY_NAME)"

# Сборка сразу под весь Unix-стек для GitHub
build-all: clean build-linux build-freebsd
	@echo " [🎉 SUCCESS] Все бинарники успешно скомпилированы и готовы к отправке на GitHub!"

# Очистка папки сборки от старых бинарников
clean:
	@echo " [🧹 CLEAN] Очистка папки bin/..."
	@rm -rf bin/
