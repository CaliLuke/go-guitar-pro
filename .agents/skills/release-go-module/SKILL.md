---
name: release-go-module
description: Release and publish this go-guitar-pro module. Use when a user asks to create, prepare, publish, verify, or troubleshoot a versioned release, GitHub release, Go proxy publication, or pkg.go.dev listing for github.com/CaliLuke/go-guitar-pro.
---

# Release Go Module

Publish one immutable semantic version from the `main` branch. Stop if a safety check fails.

## Select the version

Use the version that the user gives. Otherwise, inspect the latest tag and all changes after that tag.

- Increment the patch version for compatible fixes and documentation changes.
- Increment the minor version for new APIs, dependency updates, or breaking changes before `v1.0.0`.
- Use `v1.0.0` only when the user declares the API stable.
- For `v2.0.0` and later, update the module path with the major-version suffix before release.

Use a tag in the form `vMAJOR.MINOR.PATCH`. Do not reuse, move, delete, or force-push a published tag.

## Run the preflight checks

Read the root `AGENTS.md`. Then run:

```sh
.agents/skills/release-go-module/scripts/preflight.sh vMAJOR.MINOR.PATCH
```

The script checks the repository identity, branch, module path, tree state, version, formatting, tests, vet, and lint.

After the script passes, fetch `origin`. Make sure that `HEAD` equals `origin/main`. Make sure that the version does not exist on the remote.

```sh
git fetch --prune origin
git rev-parse HEAD
git rev-parse origin/main
git ls-remote --tags origin refs/tags/vMAJOR.MINOR.PATCH
```

If `HEAD` does not equal `origin/main`, push committed changes or stop for user direction. If the remote tag exists, do not continue.

## Prepare the release notes

Summarize user-visible changes after the previous version. State the supported Guitar Pro formats and public API changes when relevant.

Do not claim features that tests or source code do not show. Keep the notes concise. Do not include internal cleanup unless it affects users.

## Publish the GitHub release

Make sure that the GitHub repository is public. A public repository is required for the public Go services.

Create an annotated tag at the verified `HEAD`. Push only that tag. Then create a GitHub release from the tag.

```sh
git tag -a vMAJOR.MINOR.PATCH -m "vMAJOR.MINOR.PATCH"
git push origin vMAJOR.MINOR.PATCH
gh release create vMAJOR.MINOR.PATCH \
  --repo CaliLuke/go-guitar-pro \
  --title "vMAJOR.MINOR.PATCH" \
  --notes-file RELEASE_NOTES_FILE
```

Do not retag after this step. If the release notes are incorrect, edit the GitHub release without moving the tag.

## Publish to the Go proxy

Request the exact version through the public Go proxy from outside the module directory.

```sh
release_cache_dir="$(mktemp -d)"
cd /tmp
env GOPROXY=https://proxy.golang.org \
  GOMODCACHE="$release_cache_dir/mod" \
  GOCACHE="$release_cache_dir/build" \
  go list -m -json github.com/CaliLuke/go-guitar-pro@vMAJOR.MINOR.PATCH
```

Make sure that the returned `Version`, `Origin.Hash`, and `Origin.Ref` match the release. This request publishes the version to the Go module mirror.

Do not change a version after the proxy accepts it. Publish a new version if released code has an error.

## Verify pkg.go.dev

Check `https://pkg.go.dev/github.com/CaliLuke/go-guitar-pro@vMAJOR.MINOR.PATCH`. pkg.go.dev reads new versions from the Go module index every few minutes.

If the page returns 404, wait 30 seconds and check again. Stop after 10 minutes and report that indexing is pending. Do not move the tag.

## Report the result

Report these items:

- The version and commit hash.
- The GitHub release URL.
- The Go proxy result.
- The pkg.go.dev URL and indexing status.
- The preflight check results.

Leave the working tree clean. Do not create a release commit that changes only the version because this module has no version file.
