GOARCH := $(GOARCH)
GOOS := $(GOOS)
FLAGS := -ldflags "-s -w"

all:
	GOARCH=$$GOARCH GOOS=$$GOOS go build $(FLAGS) ./cmd/yggdrasil
	GOARCH=$$GOARCH GOOS=$$GOOS go build $(FLAGS) ./cmd/yggdrasilctl

clean:
	$(RM) yggdrasil yggdrasil.exe yggdrasilctl yggdrasilctl.exe

.PHONY: all clean
