# Resource Registry (Initial)

This registry tracks known resource types (FourCC), decode status, and safety tier.

## Status

- Initial scaffold created.
- Definitive list pending corpus + reference deep-dive.

## Table (to be expanded)

| FourCC | Description                                | Decode | Encode | Safety |
| ------ | ------------------------------------------ | -----: | -----: | ------ |
| TBD    | To be populated from references and corpus |     No |     No | Opaque |

## Method

1. Extract resource type set from corpus via parser.
2. Cross-check against `pylabview` and `pylavi` handling.
3. Track version-specific behavior.
