module crawlseek

go 1.16

require (
	crawlexec v0.1.0
	github.com/google/uuid v1.3.0 // indirect
	github.com/ipfs/go-ipfs-api v0.7.0 // indirect
	github.com/redis/go-redis/v9 v9.11.0 // indirect
	github.com/whyrusleeping/tar-utils v0.0.0-20180509141711-8c6c8ba81d5c // indirect
)

replace crawlexec => ./seeker/crawlexec
