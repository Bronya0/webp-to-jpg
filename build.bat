go mod tidy
go build -ldflags="-s -w" compress.go
upx -9 compress.exe