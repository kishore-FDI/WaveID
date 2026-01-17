module shazam-client

go 1.25.5

require (
	github.com/gordonklaus/portaudio v0.0.0-20230709114228-aafa478834f5
	github.com/mattn/go-sqlite3 v1.14.33
	shazam v0.0.0
)

replace shazam => ../server
