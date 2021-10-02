NAME=$(lastword $(subst /, ,$(abspath .)))
VERSION=$(shell git.exe describe --tags)
GOOPT=-ldflags "-s -w -X main.version=$(VERSION)"
ifeq ($(OS),Windows_NT)
    SHELL=CMD.EXE
    SET=set
    TYPE=type
    DEL=del
    D=$\\
else
    SET=export
    TYPE=cat
    DEL=rm
    D=/
endif

all:
	cd internal$(D)argf  && go fmt
	cd internal$(D)large && go fmt
	cd internal$(D)nonblock  && go fmt
	cd internal$(D)encoding  && go fmt
	go fmt
	$(SET) "CGO_ENABLED=0" && go build $(GOOPT)

test:
	go test -v

package:
	$(SET) "GOOS=windows" && \
	$(SET) "CGO_ENABLED=0" && \
	$(foreach GOARCH,386 amd64,\
	    $(SET) "GOARCH=$(GOARCH)" && \
	    go build -o $(NAME).exe $(GOOPT) && \
	    zip -9 $(NAME)-$(VERSION)-windows-$(GOARCH).zip $(NAME).exe && ) :
	$(SET) "GOOS=linux" && \
	$(SET) "CGO_ENABLED=0" && \
	$(foreach GOARCH,386 amd64,\
	    $(SET) "GOARCH=$(GOARCH)" && \
	    go build -o $(NAME) $(GOOPT) && \
	    tar zcvf $(NAME)-$(VERSION)-linux-$(GOARCH).tar.gz $(NAME) && ) :

clean:
	$(DEL) *.zip *.tar.gz $(NAME) $(NAME).exe
