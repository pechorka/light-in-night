illuminate-game-jam: 
	go build -o illuminate-game-jam

.PHONY: illuminate-game-jam.exe
illuminate-game-jam.exe: 
	CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -H=windowsgui" -o illuminate-game-jam.exe

windows: illuminate-game-jam.exe