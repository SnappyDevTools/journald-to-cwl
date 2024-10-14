.PHONY: build

build-go:
	go build -o build/journald-to-cwl snappydevtools.com/journald-to-cwl

build-rpm: build-go
	cp build/journald-to-cwl ./rpmbuild/SOURCES
	rpmbuild -bb --define "_topdir ${PWD}/rpmbuild" ./rpmbuild/SPECS/journald-to-cwl.spec

test:
	go test -timeout 30s -race -vet=all -v -count=1 snappydevtools.com/journald-to-cwl/...
