cp go.mod go.mod1
cp go.sum go.sum1
go test ./...
mv go.mod1 go.mod
mv go.sum1 go.sum
