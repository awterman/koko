# Koko
Koko is a cache manager. It implements strategies used in cache, e.g. `read through`.

## Example
[example.go](example/example.go)

## Introduction
Koko works with **Cache drivers** and **Callbacks**.
- **Cache drivers** wraps caches for operations like read and write. It's easy to implement a driver. Koko now provides simple driver for redis and sync.Map.
- **Callbacks** is called in situations like cache missing. With cache missing, you need to provide data for cache.