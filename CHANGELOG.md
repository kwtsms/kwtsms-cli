# Changelog

All notable changes to kwtsms-cli will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.7] - 2026-11-07

### Changed

- `send --test` now says the message was queued and not delivered.
- Package author and license holder are now kwtSMS.

## [1.0.6] - 2026-10-29

### Changed
- The repository moved to the kwtSMS GitHub organization.

## [1.0.0] - 2026-03-13

### Added
- Initial release of kwtsms-cli
- Commands: `setup`, `balance`, `senderid`, `coverage`, `send`
- Pre-built binaries for Linux, macOS and Windows
- Interactive setup wizard
- Environment variable and flag overrides
- JSON output mode via `--json`
- Test mode via `--test`
- Bulk send
- Send log
- Arabic-Indic digit conversion and phone number normalisation
- `--version` flag
