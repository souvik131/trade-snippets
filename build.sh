rm -rf build;
mkdir build;
go build -o mac .;
env GOOS="windows" GOARCH="amd64" CGO_ENABLED="1" CC="x86_64-w64-mingw32-gcc" go build -o win.exe .;
env GOOS="linux" GOARCH="arm" go build -o linux .;
cp mac build/;
cp win.exe build/;
cp linux build/;
rm mac
rm linux
rm win.exe
# rm build.zip
# zip -r build.zip build;
# rm -rf build;


