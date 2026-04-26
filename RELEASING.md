# Releasing go-service-kit

This repository should use tagged releases. Consumers should depend on tags such as `v0.1.0`, not floating commits, unless they are intentionally testing unreleased changes.

## Release checklist

1. Run formatting and tests:

   ```powershell
   gofmt -w ./...
   go test ./...
   ```

2. Review the public package surface in `README.md`.
3. Update `CHANGELOG.md`.
4. Add or update the release note file for the version being cut.
5. Commit the release contents.
6. Create an annotated tag:

   ```powershell
   git tag -a v0.1.0 -m "go-service-kit v0.1.0"
   ```

7. Push the branch and tag:

   ```powershell
   git push origin main
   git push origin v0.1.0
   ```

8. Publish the GitHub release using the corresponding release note.

## Versioning

- use `v0.x.y` while the API is still changing during adoption
- move to `v1.0.0` only after the package surface has settled across multiple services

## Release artifacts

Each release should have:

- a git tag
- a `CHANGELOG.md` entry
- a release note markdown file
- updated README usage guidance when the public package surface changes

