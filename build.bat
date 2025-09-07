go mod tidy
go build -ldflags="-s -w" -o compress.exe main.go
upx -9 compress.exe