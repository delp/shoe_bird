GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CXX=x86_64-w64-mingw32-g++ CC=x86_64-w64-mingw32-gcc go build

mkdir winbuild
7za a winbuild/thebird.7z sheet.csv sheet.png shoe_bird.exe
