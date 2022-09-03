build:
	go build -ldflags="-w -s" 

install:
	mv xgit /usr/bin/xgit

termux-install:
	mv xgit /data/data/com.termux/files/usr/bin/xgit

all: build install

termux-all: build termux-install

clean:
	rm -f xgit /usr/bin/xgit

termux-clean:
	rm -f xgit /data/data/com.termux/files/usr/bin/xgit
