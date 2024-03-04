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

export DATABASE_DSN := postgres://url-shortener:url-shortener@localhost:5432/url-shortener?sslmode=disable
env:
	echo "export DATABASE_DSN=$(DATABASE_DSN)" > .$@

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
	@cd test && ./shortenertestbeta -test.v -test.run=^TestIteration1$$ -binary-path=../build/shortener/shortener
	@cd test && ./shortenertestbeta -test.v -test.run=^TestIteration2$$ -source-path=../
	@cd test && ./shortenertestbeta -test.v -test.run=^TestIteration6$$ -source-path=../ -binary-path=../build/shortener/shortener
	@cd test && ./shortenertestbeta -test.v -test.run=^TestIteration11$$ -binary-path=../build/shortener/shortener -database-dsn='$(DATABASE_DSN)'
	@cd test && ./shortenertestbeta -test.v -test.run=^TestIteration13$$ -binary-path=../build/shortener/shortener -database-dsn='$(DATABASE_DSN)'
	@cd test && ./shortenertestbeta -test.v -test.run=^TestIteration14$$ -binary-path=../build/shortener/shortener -database-dsn='$(DATABASE_DSN)'

YA := https://ya.ru
GOOGLE := https://www.google.com
SVC := http://localhost:8080
COOKIE := /tmp/cookie.txt
ID=Alexey

curl:
	@curl --cookie "id=$(ID)" -X POST -d "url=$(YA)" $(SVC); echo
	@curl -X POST -d "url=$(GOOGLE)" $(SVC); echo
	#@curl --cookie "id=$(ID)" -X POST -H "Content-Type: application/json" -d '{"url":"$(YA)"}' $(SVC)/api/shorten; echo

list:
	@curl --cookie "id=$(ID)" $(SVC)/api/user/urls; echo

delete:
	echo '["821GuQ"]' | curl -X DELETE -H "Content-Type: application/json" --data-binary @- --cookie "id=$(ID)" $(SVC)/api/user/urls; echo

gzip:
	@echo '{"url":"$(YA)"}' | gzip | curl -v -i --data-binary @- -H "Content-Type: application/json" -H "Content-Encoding: gzip" $(SVC)/api/shorten; echo

batch:
	@echo '[{"correlation_id":"1","original_url":"http://vz.ru"}]' | curl --cookie "id=$(ID)" -X POST -v -i --data-binary @- -H "Content-Type: application/json" $(SVC)/api/shorten/batch; echo
	@echo '[{"correlation_id":"1","original_url":"$(YA)"},{"correlation_id":"2","original_url":"$(GOOGLE)"}]' | \
		curl --cookie "id=$(ID) "-X POST -v -i --data-binary @- -H "Content-Type: application/json" $(SVC)/api/shorten/batch; echo

conflict:
	@curl -X POST -H "Content-Type: application/json" -d '{"url":"$(YA)"}' $(SVC)/api/shorten; echo
	@curl -X POST -H "Content-Type: application/json" -d '{"url":"$(YA)"}' $(SVC)/api/shorten; echo

up:
	sudo docker compose up -d
	while ! pg_isready -q -h localhost; do true; done
	#psql "$(DATABASE_DSN)" -f db/urls_table.sql

down:
	sudo docker compose down

.PHONY: clean
clean:
	rm -rvf $(BUILD)
	rm -rvf test/
