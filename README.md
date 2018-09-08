# dandelion

A configuration publish system build on the top of git filesystem.

## Installation

### Server

```sh
go get -u github.com/tengattack/dandelion/cmd/dandelion
```

1. Import `data/schema.sql` to a mysql database.
2. Copy and modify `cmd/dandelion/config.example.yml` to `/etc/dandelion/config.yml`.
3. Run `dandelion -config /etc/dandelion/config.yml`

### Client

```sh
go get -u github.com/tengattack/dandelion/cmd/dandelion-seed
```

1. Copy and modify `cmd/dandelion-seed/config.example.yml` to `/etc/dandelion-seed/config.yml`.
2. Run `dandelion-seed -config /etc/dandelion-seed/config.yml`

## WebUI

If you need to modify web ui, as following steps:

```sh
cd web
npm i
npm run clean
npm run build
# return to repository root path
cd ..
# regenerate bindata.go file
go generate ./...
cd cmd/dandelion
go install
```

## License

MIT
