linux: 
	go build -o light-in-night

app-amd64.syso:
	rsrc -arch amd64 -ico app.ico -o app-amd64.syso

.PHONY: light-in-night.exe
light-in-night.exe: app-amd64.syso
	CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -H=windowsgui" -o light-in-night.exe

windows: light-in-night.exe