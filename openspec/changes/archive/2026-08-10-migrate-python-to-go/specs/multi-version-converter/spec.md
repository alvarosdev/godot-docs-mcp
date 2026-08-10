## Purpose

Download Godot Engine documentation from the official repository, convert RST files to Markdown, and produce a clean versioned artifact ready for the MCP server to serve.

## ADDED Requirements

### Requirement: Download Godot docs from GitHub

The pipeline SHALL download documentation for specified Godot versions as ZIP archives from `https://github.com/godotengine/godot-docs` and extract them into versioned subdirectories under `docs/{version}/`.

#### Scenario: Single version download
- **WHEN** the pipeline is invoked with version `"4.7"`
- **THEN** the Godot 4.7 docs branch is downloaded and extracted to `docs/4.7/`

#### Scenario: Multiple versions
- **WHEN** the pipeline is invoked with versions `"3.6,4.7"`
- **THEN** both versions are downloaded into `docs/3.6/` and `docs/4.7/` respectively

### Requirement: RST to Markdown conversion

The pipeline SHALL convert all `.rst` files in the downloaded documentation to `.md` files using pandoc, running conversions in parallel across available CPU cores.

#### Scenario: Parallel conversion
- **WHEN** the docs directory contains 100 `.rst` files
- **THEN** all files are converted to `.md` using all available CPU cores
- **AND** a summary reports the number of successful and failed conversions

#### Scenario: No RST files present
- **WHEN** the docs directory contains no `.rst` files
- **THEN** the pipeline skips conversion and reports zero files converted

### Requirement: Cleanup non-Markdown files

The pipeline SHALL remove all files that are not `.md` or `docs_tree.txt` from the versioned docs directories, and remove any resulting empty directories.

#### Scenario: Post-conversion cleanup
- **WHEN** conversion is complete and `.rst` source files remain alongside `.md` output files
- **THEN** all non-`.md` files are deleted
- **AND** empty directories are removed

### Requirement: Generate documentation tree

The pipeline SHALL generate a `docs_tree.txt` file containing a tree-style directory listing at the root of each version's docs directory.

#### Scenario: Tree generation
- **WHEN** the pipeline completes for version `4.7`
- **THEN** `docs/4.7/docs_tree.txt` exists and contains a tree listing of the directory structure

### Requirement: Generate versions manifest

The pipeline SHALL write `docs/versions.json` containing the list of successfully processed versions, the latest version identifier, and the build date.

#### Scenario: Multi-version manifest
- **WHEN** versions `3.6` and `4.7` are successfully processed
- **THEN** `docs/versions.json` contains `{"versions": ["3.6", "4.7"], "latest": "4.7", "built": "YYYY.MM.DD"}`
- **AND** versions are sorted in ascending order

#### Scenario: Single version manifest
- **WHEN** only version `4.7` is processed
- **THEN** `latest` is `"4.7"` and `versions` contains only `["4.7"]`
