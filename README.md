# nbt-cli

Minecraft NBT map editor for block entities.

## Build

```
make build
```

## Usage

```
./bin/nbt-cli map get --region-dir <path> --x <int> --y <int> --z <int> [--print-region]
```

Pass `--region-file` instead of `--region-dir` to target a single region file. Use `map create` or `map delete` for CRUD operations.


