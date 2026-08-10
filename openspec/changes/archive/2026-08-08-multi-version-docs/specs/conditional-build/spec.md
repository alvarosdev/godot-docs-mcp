## Purpose

Ensures the Docker image is only rebuilt when the upstream Godot documentation has actually changed, using a SHA-based lockfile to detect changes without downloading content.

## ADDED Requirements

### Requirement: Image is only rebuilt when docs change
The CI workflow SHALL compare the HEAD SHA of each tracked Godot docs branch against a `versions.lock` file in the repository. The build SHALL be skipped if no SHA has changed since the last successful build.

#### Scenario: No changes detected
- **WHEN** the scheduled workflow runs and all branch HEAD SHAs match the values in `versions.lock`
- **THEN** the build job is skipped and no image is pushed

#### Scenario: One version branch changed
- **WHEN** the 4.7 branch HEAD SHA differs from the value in `versions.lock`
- **THEN** a full build is triggered, a new image is pushed, and `versions.lock` is updated with the new SHA

#### Scenario: New major version branch appears
- **WHEN** a new major version branch (e.g., `5.0`) appears that has no entry in `versions.lock`
- **THEN** a full build is triggered including the new version

#### Scenario: First build (no lockfile)
- **WHEN** `versions.lock` does not exist in the repository
- **THEN** a full build is triggered unconditionally

### Requirement: versions.lock is committed on successful build
After a successful build and push, the workflow SHALL update `versions.lock` with the current HEAD SHA of each included version branch and push the change to the repository.

#### Scenario: Build succeeds
- **WHEN** the image is built and pushed successfully for versions 3.6 and 4.7
- **THEN** `versions.lock` is updated with the current SHAs and committed back to the default branch

#### Scenario: Build fails
- **WHEN** the image build or push fails
- **THEN** `versions.lock` is NOT updated, ensuring the next scheduled run will retry with the same SHAs

### Requirement: Image is tagged with year-month format
Pushed images SHALL be tagged with a `YYYY.MM.DD` format tag derived from the build date, plus a `latest` tag.

#### Scenario: Scheduled build on August 3, 2026
- **WHEN** the workflow builds on August 3, 2026
- **THEN** the image is pushed with tags `2026.08.03` and `latest`

#### Scenario: Manual dispatch build
- **WHEN** the workflow is triggered manually via `workflow_dispatch`
- **THEN** the same tagging scheme applies: `YYYY.MM.DD` from the build date plus `latest`

### Requirement: version selection picks latest minor per major
The workflow SHALL fetch all `major.minor` branches from the godot-docs repository, group them by major version, and select only the highest minor within each group.

#### Scenario: Standard selection
- **WHEN** the godot-docs repository has branches `3.5`, `3.6`, `4.6`, `4.7`
- **THEN** the build matrix is `["3.6", "4.7"]`

#### Scenario: Workflow dispatch with explicit version
- **WHEN** the workflow is manually triggered with `godot_version=3.6`
- **THEN** only version `3.6` is built, overriding automatic selection

### Requirement: SHA comparison uses GitHub API only
The change detection SHALL use only the GitHub REST API to fetch branch references. No git clone or zip download SHALL be performed during the detection phase.

#### Scenario: Detection phase completes quickly
- **WHEN** checking 2 tracked versions
- **THEN** the detection phase completes in under 5 seconds without downloading any repository content
