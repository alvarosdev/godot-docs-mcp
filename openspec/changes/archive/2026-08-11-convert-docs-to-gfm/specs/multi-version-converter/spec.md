## MODIFIED Requirements

### Requirement: RST to Markdown conversion

The pipeline SHALL convert all `.rst` files in the downloaded documentation to GitHub-Flavored Markdown (`.md`) files using pandoc, running conversions in parallel across available CPU cores. Sphinx `.. table::` directives SHALL be stripped before conversion to enable grid-table rendering. On successful conversion, the source `.rst` SHALL be deleted. On failed conversion, the `.rst` SHALL be preserved as a fallback.

#### Scenario: Parallel conversion
- **WHEN** the docs directory contains 100 `.rst` files
- **THEN** all files are converted to `.md` using all available CPU cores
- **AND** a summary reports the number of successful and failed conversions

#### Scenario: No RST files present
- **WHEN** the docs directory contains no `.rst` files
- **THEN** the pipeline skips conversion and reports zero files converted

#### Scenario: Successful conversion deletes source
- **WHEN** a `.rst` file converts successfully to `.md`
- **THEN** the source `.rst` file is deleted
- **AND** only the `.md` output remains

#### Scenario: Failed conversion preserves RST fallback
- **WHEN** a `.rst` file fails to convert (pandoc error, malformed RST)
- **THEN** the original `.rst` file is preserved alongside any partial output
- **AND** the failure is reported in the conversion summary

#### Scenario: Sphinx table directive stripped
- **WHEN** a `.rst` file contains a `.. table::` directive
- **THEN** the directive and its `:widths:` option are removed before pandoc conversion
- **AND** the resulting `.md` contains a legible Markdown grid table

### Requirement: Cleanup non-Markdown files

The pipeline SHALL remove all files that are not `.md`, `.rst`, or `docs_tree.txt` from the versioned docs directories, and remove any resulting empty directories. The `.rst` files SHALL be preserved only when their conversion failed; successfully-converted `.rst` files are already deleted by the conversion step.

#### Scenario: Post-conversion cleanup
- **WHEN** conversion is complete and source `.rst` files remain
- **THEN** the remaining `.rst` files are only those whose conversion failed
- **AND** empty directories are removed
