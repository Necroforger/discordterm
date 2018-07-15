H="$(pwd)"
cd cmd/discordterm
export GOOS=windows
export GOARCH=386
go build
cd "$H"
