BINARY  := claude-statusline
INSTALL := ~/.claude/$(BINARY)
LDFLAGS := -s -w

.PHONY: build install clean

build:
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) .

install: build
	cp -f $(BINARY) $(INSTALL)

clean:
	rm -f $(BINARY)
