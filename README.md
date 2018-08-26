# dandelion

A configuration publish system build on the top of git filesystem.

## Installation

1. Import `data/schema.sql` to a mysql database.
2. Copy and modify `cmd/dandelion/config.example.yml` to `/etc/dandelion/config.yml`.
3. Run `dandelion -config /etc/dandelion/config.yml`

## Usage (Client)

1. Copy and modify `cmd/dandelion-seed/config.example.yml` to `/etc/dandelion-seed/config.yml`.
2. Run `dandelion-seed -config /etc/dandelion-seed/config.yml`

## License

MIT
