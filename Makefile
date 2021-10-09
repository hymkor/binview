NAME=$(lastword $(subst /, ,$(abspath .)))
VERSION=$(shell git.exe describe --tags)
GOOPT=-ldflags "-s -w -X main.version=$(VERSION)"
ifeq ($(OS),Windows_NT)
    SHELL=CMD.EXE
    SET=set
    DEL=del
else
    SET=export
    DEL=rm
endif

all:
	cd internal/argf     && go fmt
	cd internal/large    && go fmt
	cd internal/nonblock && go fmt
	cd internal/encoding && go fmt
	go fmt
	$(SET) "CGO_ENABLED=0" && go build $(GOOPT)

test:
	go test -v

_package_windows:
	$(SET) "GOOS=windows" && \
	$(SET) "CGO_ENABLED=0" && \
	go build $(GOOPT) && \
	zip -9 $(NAME)-$(VERSION)-windows-$(GOARCH).zip $(NAME).exe

_package_linux:
	$(SET) "GOOS=linux" && \
	$(SET) "CGO_ENABLED=0" && \
	go build $(GOOPT) && \
	tar zcvf $(NAME)-$(VERSION)-linux-$(GOARCH).tar.gz $(NAME)

package:
	$(SET) "GOARCH=386"   && $(MAKE) _package_windows
	$(SET) "GOARCH=amd64" && $(MAKE) _package_windows
	$(SET) "GOARCH=386"   && $(MAKE) _package_linux
	$(SET) "GOARCH=amd64" && $(MAKE) _package_linux

clean:
	$(DEL) *.zip *.tar.gz $(NAME) $(NAME).exe
