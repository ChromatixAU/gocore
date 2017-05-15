PACKAGES=github.com/chromatixau/negroni github.com/chromatixau/render

all: build

build:
	GOPATH=$(PWD) go build

install: clean
	GOPATH=$(PWD) go get $(PACKAGES)

clean:
	rm -rf src/github.com
