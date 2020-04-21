GOARCH := $(GOARCH)
GOOS := $(GOOS)
FLAGS := -ldflags "-s -w"

all:
	GOARCH=$$GOARCH GOOS=$$GOOS go build $(FLAGS) ./cmd/popura

clean:
	$(RM) popura popura.exe

.PHONY: all clean
