# Platform Bindings

This directory contains bindings to Replicant for various platforms. These are high-level,
idiomatic, platform-specific bindings that app developers use.

Each binding typically calls through to `repm` or `api` to do the actual work, but those
packages are too low-level to be useful to application developers.
