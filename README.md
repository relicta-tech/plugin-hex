# Hex Plugin for Relicta

Official Hex plugin for [Relicta](https://github.com/relicta-tech/relicta) - Publish packages to Hex.pm (Elixir).

## Installation

```bash
relicta plugin install hex
relicta plugin enable hex
```

## Configuration

Add to your `release.config.yaml`:

```yaml
plugins:
  - name: hex
    enabled: true
    config:
      # Add configuration options here
```

## License

MIT License - see [LICENSE](LICENSE) for details.
