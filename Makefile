BUILD := build
CMD := ./cmd
APP := shortener
MODULE := github.com/aleks0ps/url-shortener

.PHONY: all
all: build

go.mod:
	go mod init $(MODULE)

.PHONY: build
build: go.mod
	mkdir -vp $(BUILD)/$(APP)
	mkdir -vp $(BUILD)/client
	# use local path, otherwise 'go build' will lookup global dir /usr/local/go/src/cmd/ 
	go build -o $(BUILD)/$(APP)/$(APP) $(CMD)/$(APP)
	go build -o $(BUILD)/client/client $(CMD)/client

SERVER_PORT := 8080
export SERVER_PORT

.PHONY: test
test: build
	@mkdir -vp test
	@if ! test -f test/shortenertest; then \
	  wget -P test/ https://github.com/Yandex-Practicum/go-autotests/releases/download/v0.10.3/shortenertest; \
	  chmod +x test/shortenertest; \
	fi
	if ! test -f test/shortenertestbeta; then \
	  wget -P test/ https://github.com/Yandex-Practicum/go-autotests/releases/download/v0.10.3/shortenertestbeta; \
	  chmod +x test/shortenertestbeta; \
	fi
	@cd test && ./shortenertest -test.v -test.run=^TestIteration1$$ -binary-path=../build/shortener/shortener
	@cd test && ./shortenertest -test.v -test.run=^TestIteration2$$ -source-path=../
	@cd test && ./shortenertest -test.v -test.run=^TestIteration3$$ -source-path=../
	@cd test && ./shortenertest -test.v -test.run=^TestIteration1$$ -binary-path=../build/shortener/shortener -server-port=$$SERVER_PORT
	@cd test && ./shortenertestbeta -test.v -test.run=^TestIteration6$$ -source-path=../ -binary-path=../build/shortener/shortener

YA := https://ya.ru
GOOGLE := https://www.google.com
SVC := http://localhost:8080
 
curl:
	curl -X POST -d "url=$(YA)" $(SVC)
	@echo
	curl -X POST -d "url=$(GOOGLE)" $(SVC);
	@echo
	curl -X POST -H "Content-Type: application/json" -d '{"url":"$(YA)"}' $(SVC)/api/shorten 
	@echo

.PHONY: clean
clean:
	rm -rvf $(BUILD)
	rm -rvf test/
