# Platformer

# Linux Build

```bash
go build
```

## Windows Build
```bash
sudo apt-get install gcc-mingw-w64
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CXX=x86_64-w64-mingw32-g++ CC=x86_64-w64-mingw32-gcc go build
```
