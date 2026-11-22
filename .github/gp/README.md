# Glitchedpixel-io's patched tinyauth

This fork exists to run one fix that upstream has not shipped yet. It is not a product of its
own. **Retire it when upstream releases a fix for
[tinyauthapp/tinyauth#1216](https://github.com/tinyauthapp/tinyauth/issues/1216)**: switch
Core back to `ghcr.io/tinyauthapp/tinyauth`, then archive this repository.

## What is patched

`gp/v5.2.0` is upstream's `v5.2.0` tag plus the commits below. `main` mirrors upstream and
is never built.

| Change | Why |
|---|---|
| A password is checked on a connection of its own (`LdapService.CheckPassword`), and the shared LDAP connection is never re-bound | Every basic-auth login bound the shared connection as the user until the service account was re-bound. A search from another request in that gap ran as the user, who in LLDAP sees only their own entry. That search found nothing: basic-auth logins got 401s (`multiple or no entries found`) and signed-in browsers got a 302 to the login page. On Core on 2026-11-22, a dashboard tab was refused about 1 read in 25. |

The regression test is `internal/service/ldap_service_integration_test.go`, run against a
real LLDAP by `.github/workflows/gp-image.yml`. On stock v5.2.0 it fails: 400 of 800
searches came back empty.

## Building and releasing

- A PR into `gp/v5.2.0` runs the `test` job: upstream's unit tests and the LLDAP
  integration test.
- A push to a `gp/*` branch also builds `ghcr.io/glitchedpixel-io/tinyauth:<VERSION>`, with
  `<VERSION>` read from `.github/gp/VERSION`, for example `5.2.0-gp.1`. A published tag is
  never overwritten, so each change bumps `VERSION` in the same PR.
- Upstream's own workflows are deleted on `gp/*` branches and disabled on `main`. Its
  `release.yml` would otherwise publish to this org's GHCR whenever a `v*` tag was pushed,
  and its `ci.yml` would run on every PR and fail on its Codecov upload.
- Step 1 below re-adds upstream's workflow files to the new branch, so delete them again
  in the cherry-pick PR.

## Taking a new upstream release

1. `git fetch upstream --tags`, then push the new tag as a fresh branch:
   `git push origin refs/tags/vX.Y.Z:refs/heads/gp/vX.Y.Z`.
2. Check upstream's notes for #1216. If it is fixed, stop here and retire the fork.
3. Cherry-pick this branch's commits onto `gp/vX.Y.Z` in a PR, setting `VERSION` to
   `X.Y.Z-gp.1`.
4. Change the image in Core's tinyauth compose file
   (`/mnt/user/appdata/tinyauth/docker-compose.yml`), then run `docker compose up -d tinyauth`.

## Licence

tinyauth is AGPL-3.0. Anyone who signs in through the instance this builds is interacting with
it over a network, and this public repository is where its modified source is offered to them.
