# GITXPOZ

Found exposed git repositories.

## Usage:

-h, --help                   Show this help
-t, --thread=NBR             Number of threads (default 50)
-o, --output=FILE            Output file (default found_git.txt)
-i, --input=FILE             Input file
-k, --insecure               Ignore certificate errors
-p, --proxy=PROXY            Use proxy (proto://ip:port)
-V, --version                Print version and exit

## Examples:

```sh
$ gitXpoz -i top-alexa.txt
$ gitXpoz -p socks5://127.0.0.1:9050 -o good.txt -i top-alexa.txt -t 60
```

## Install:

With one liner if **$GOROOT/bin/** is in **$PATH**:

```sh
go install github.com/mmpx12/gitXpoz@latest
```

or from source with:

```sh
git clone https://github.com/mmpx12/gitXpoz.git
cd gitXpoz
make
sudo make install
# or 
sudo make all
```

for **termux** you can do:

```sh
git clone https://github.com/mmpx12/gitXpoz.git
cd gitXpoz
make
make termux-install
# or
make termux-all
```


There is also prebuild binaries [here](https://github.com/mmpx12/gitXpoz/releases/latest).
