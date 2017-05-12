PACKAGES=github.com/urfave/negroni github.com/unrolled/render github.com/chromatixau/gomiddlware

all: build

build:
	GOPATH=$(PWD) go build

install: clean
	GOPATH=$(PWD) go get $(PACKAGES)

clean:
	rm -rf src/github.com
