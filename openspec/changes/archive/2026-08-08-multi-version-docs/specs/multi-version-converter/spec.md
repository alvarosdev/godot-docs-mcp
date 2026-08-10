## Purpose

Downloads and converts multiple Godot documentation versions into a versioned directory structure, producing per-version markdown files and a shared metadata manifest.

## ADDED Requirements

### Requirement: Converter accepts a list of versions
The converter SHALL accept a list of Godot version strings (e.g., `["3.6", "4.7"]`) and process each version sequentially.

#### Scenario: Multiple versions provided
- **WHEN** the converter is invoked with versions `["3.6", "4.7"]`
- **THEN** it downloads and converts docs for version 3.6 first, then version 4.7

#### Scenario: Single version provided
- **WHEN** the converter is invoked with versions `["4.7"]`
- **THEN** it downloads and converts docs only for version 4.7

#### Scenario: Empty version list
- **WHEN** the converter is invoked with an empty version list
- **THEN** it exits early with an error message and non-zero exit code

### Requirement: Docs are stored in versioned subdirectories
Each version's converted documentation SHALL be stored under `docs/{version}/`, maintaining the original internal directory structure.

#### Scenario: Two versions converted
- **WHEN** versions 3.6 and 4.7 are both converted successfully
- **THEN** the output directory contains `docs/3.6/classes/`, `docs/3.6/tutorials/`, `docs/4.7/classes/`, and `docs/4.7/tutorials/`

#### Scenario: Per-version tree file
- **WHEN** a version is converted
- **THEN** a `docs_tree.txt` file is generated inside that version's directory (e.g., `docs/4.7/docs_tree.txt`)

### Requirement: versions.json is generated after all versions are processed
After all versions are converted, the converter SHALL write a `docs/versions.json` file containing the list of available versions, which version is latest, and the build date.

#### Scenario: Successful multi-version build
- **WHEN** versions `["3.6", "4.7"]` are both converted successfully
- **THEN** `docs/versions.json` contains `{"versions": ["3.6", "4.7"], "latest": "4.7", "built": "YYYY.MM.DD"}` where the latest is the highest version and built reflects the current date

#### Scenario: Single version is both available and latest
- **WHEN** only version `["4.7"]` is converted
- **THEN** `docs/versions.json` contains `{"versions": ["4.7"], "latest": "4.7", "built": "YYYY.MM.DD"}`

### Requirement: Conversion failure for one version does not block others
If conversion fails for one version, the converter SHALL report the error and continue processing remaining versions. The failed version SHALL NOT appear in `versions.json`.

#### Scenario: One version fails, the other succeeds
- **WHEN** version 3.6 fails during download or conversion and version 4.7 succeeds
- **THEN** `docs/4.7/` exists with converted docs, `docs/versions.json` only lists `["4.7"]`, and the converter exits with a non-zero exit code after logging the error

### Requirement: Only latest minor of each major is selected
The version resolution logic SHALL group all available `major.minor` branches by major version number and select only the highest minor within each major group.

#### Scenario: Multiple minors for the same major
- **WHEN** the godot-docs repository has branches `3.5`, `3.6`, `4.6`, `4.7`
- **THEN** only `3.6` and `4.7` are selected for download

#### Scenario: New major version appears
- **WHEN** a new major version branch appears (e.g., `5.0`)
- **THEN** it is automatically included in the version list for the next build
