# Contributing to ISOTOPE

Thank you for your interest in developing ISOTOPE. This document describes how to help the project.

---

## Code of Conduct

Participants must follow the [Code of Conduct](CODE_OF_CONDUCT.md). In short: respect, no violence, no discrimination.

---

## How to Help

### Report a Bug
- Check if the issue already exists
- Describe: what you did, what you expected, what you got
- Attach logs (without private data)
- Specify version and environment

### Suggest an Improvement
- Open an issue with the `enhancement` label
- Describe: why it's needed, how it should work
- Discuss with the community before writing code

### Write Code
1. Fork the repository
2. Create a branch: `feature/name` or `fix/name`
3. Write code and tests
4. Make sure `go test ./...` passes
5. Create a Pull Request

---

## Code Standards

- **Go:** `gofmt`, `go vet`, `golint`
- **Flutter:** `flutter analyze`, `flutter test`
- **Commits:** meaningful messages in English
- **Tests:** required for new functionality

---

## Project Structure

    isotope-core/
    ├── node/           # Go core
    │   ├── main.go     # Entry point
    │   ├── node.go     # P2P node, processMessage
    │   ├── memory.go   # Weighted memory
    │   ├── layers.go   # Neural network
    │   ├── handlers.go # HTTP + WebSocket
    │   ├── utils.go    # Utilities, vectorization
    │   └── state.go    # State persistence
    ├── mobile/         # Flutter app
    ├── tests/          # Test suites
    └── docs/           # Documentation

---

## Priorities

Current tasks are marked in the [Roadmap](docs/roadmap/ROADMAP.md) and Issues with the `help wanted` label.

---

## If GitHub Is Unavailable

If GitHub is blocked:
- Forks and PRs are accepted via mirrors (GitLab, Codeberg)
- Releases are available at isotope.network and IPFS

[See mirrors →](docs/MIRRORS.md)

---

## Contacts

- GitHub: [github.com/isotope-network](https://github.com/isotope-network)
- Website: [isotope.network](https://isotope.network)

---

**ISOTOPE is built by the community. Every contribution matters.**