PLATFORMS= freebsd/amd64 linux/amd64 linux/arm-6 linux/arm-7 linux/arm64 windows/386 windows/amd64 darwin/amd64
AMTC_CHECKOUT=amtc
AMTCWEB_CHECKOUT=$(AMTC_CHECKOUT)/amtc-web

VERSION=$(shell git describe --tags | cut -dv -f2)
LDFLAGS:=-X main.AppVersion=$(VERSION) -w
GOBIN ?= $(CWD)
BIN=$(shell echo amtgo-`uname -s`-`uname -m`)
SOURCES=$(shell echo main.go ./*/*.go)

$(BIN): $(SOURCES) dependencies
	go build -v -ldflags "$(LDFLAGS) -s" -o "$(BIN)" main.go

zip: $(BIN) $(BIN)_v$(VERSION).zip

$(BIN)_v$(VERSION).zip:
	zip $(BIN)_v$(VERSION).zip $(BIN)

dependencies:
	go get -v
	touch dependencies

release: xgo dependencies
	xgo -go 1.8.x -ldflags "$(LDFLAGS)" -targets "$(PLATFORMS)" .

ziprelease: release
	for bin in amtgo-*; do zip $${bin}_v$(VERSION).zip $$bin; done

xgo:
	go get github.com/karalabe/xgo

lint:
	golint ./...

test: dependencies
	go test -v -coverprofile=webserver.out ./webserver
	go test -v -coverprofile=database.out ./database
	go vet ./...

coverage: test
	# https://github.com/golang/go/issues/6909
	go tool cover -html=webserver.out
	go tool cover -html=database.out

# for codecov.io
codecov.io: dependencies
	echo "" > coverage.txt
	for d in $(shell go list ./... | grep -v vendor); do \
		go test -coverprofile=profile.out -covermode=atomic $$d; \
		[ -f profile.out ] && cat profile.out >> coverage.txt && rm profile.out; \
	done

# targets below only required to rebuild assets

go-bindata:
	go get github.com/jteeuwen/go-bindata/...

$(AMTC_CHECKOUT):
	git clone https://github.com/schnoddelbotz/amtc.git
	# patch only bit in amtc-web: initial setup dialogue
	cp webserver/setup.hbs.html amtc/amtc-web/templates/
	perl -pi -e "s@localhost@@" amtc/amtc-web/js/app/app.js
	perl -pi -e "s@mysqlUser: 'amtcweb'@mysqlUser: ''@" amtc/amtc-web/js/app/app.js
	perl -pi -e "s@Uptime, load average@amtgo uptime@" amtc/amtc-web/templates/systemhealth.hbs.html
	perl -pi -e "s@Disk free@amtgo memory usage@" amtc/amtc-web/templates/systemhealth.hbs.html
	perl -pi -e "s@Active amtc processes@Active go routines@" amtc/amtc-web/templates/systemhealth.hbs.html
	perl -pi -e "s@PHP@Go@" amtc/amtc-web/templates/systemhealth.hbs.html
	perl -pi -e "s@amtc binary@amtgo binary@" amtc/amtc-web/templates/systemhealth.hbs.html
	perl -pi -e "s@\(notyet\)@@" amtc/amtc-web/templates/systemhealth.hbs.html
	cd amtc/amtc-web ; make -j 8

assets: go-bindata $(AMTC_CHECKOUT)
	mkdir -p web-assets/{js,fonts,css,page} amt-cmd
	cp $(AMTCWEB_CHECKOUT)/{index.html.gz,amtc-favicon.png} web-assets
	cp $(AMTCWEB_CHECKOUT)/css/styles.css.gz web-assets/css
	cp $(AMTCWEB_CHECKOUT)/js/jslibs.js.gz web-assets/js
	cp $(AMTCWEB_CHECKOUT)/page/*.md web-assets/page
	cp $(AMTCWEB_CHECKOUT)/fonts/fontawesome-webfont.woff* web-assets/fonts
	cp $(shell ls -1 $(AMTC_CHECKOUT)/src/wsman_* | grep -v '\.h$$' | xargs) amt-cmd
	go-bindata -pkg webserver -prefix "`pwd`/web-assets" -nocompress -nomemcopy -o webserver/assets.go web-assets/...
	go-bindata -pkg amt -prefix "`pwd`/amt-cmd" -nocompress -nomemcopy -o amt/commands.go amt-cmd/...

clean:
	rm -f $(BIN) $(BIN).zip *.out dependencies

realclean: clean
	rm -rf amtc amt-cmd amtgo-*
