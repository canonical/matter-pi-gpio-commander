# Automated snap releases and weekly canary

Routine upstream releases should need no maintainer action. `main` ships the
latest Connected Home over IP release that passes this repository's tests.
`dev` tests upstream development early; it is not a staging branch that must
be merged before every release.

| Repository branch | `connectedhomeip` source in `snap/snapcraft.yaml` | When to build | Store channels |
| --- | --- | --- | --- |
| `main` | `source-tag:` pinned to a four-part release tag | Release PR, then push to `main` | `latest/candidate` → `latest/stable` |
| `dev` | `source-branch: master` (upstream's development branch) | Weekly, or manually to verify a fix | `latest/edge` → optionally `latest/beta` |

## Release path: automatic unless the update breaks something

1. Renovate opens a PR against `main` for the latest non-prerelease
   `project-chip/connectedhomeip` release matching `vX.Y.Z.W`. Keep this update
   rule off `dev`.
2. `snap-pr.yml` checks that the release pin is still a tag, builds the arm64
   snap, and runs `.github/actions/test-snap` on the local artifact without
   Store credentials. Make `Snap PR build and test (arm64)` a required check on `main`.
   Renovate may auto-merge a passing pin update; a failed one stays open.
3. After merge, `snap-release.yml` builds the pinned `main` commit, publishes it to
   `latest/candidate`, and runs the shared suite on the Store install. Verify
   that the installed revision is the one just uploaded, then release **that
   revision** to `latest/stable`. A failed build, upload, or test leaves stable
   unchanged.

## Canary path: early warning, independent of stable

1. Create `dev` from `main` and commit `source-branch: master` in place of the
   release tag. The weekly schedule on `main` dispatches `snap-release.yml`
   on `dev`. The build and tests run in the `dev` branch context at the event
   SHA, just as for a manual dispatch on `dev`.
2. If the arm64 build succeeds, publish it to `latest/edge` and run the shared
   suite on that revision. If the build fails, edge stays unchanged. If tests
   fail, the build stays on edge for diagnosis. Surface failures through
   GitHub Actions notifications; include the local and upstream commit SHAs
   when available and retain build/test logs.
3. **Recommendation: keep `latest/beta` as the last passing canary.** On
   success, release the tested edge revision to beta; on failure, leave beta
   alone. If nobody uses beta, drop just this promotion step. The weekly test
   and failure notification are what give the canary its value.

A failed canary does not block a release PR: upstream may fix the problem
before tagging a release, or the release may not contain the breaking change.
A passing canary likewise does not replace testing the pinned release.

## When a compatibility fix is needed

Land a focused fix and, where possible, a regression test on `dev`. Keep the
fix separate from the branch's upstream-ref change so it can be cherry-picked.
Run the canary manually to verify it.

- **Works with the current release too:** cherry-pick the fix into a normal PR
  against `main`. Let Renovate handle the next pin bump as usual.
- **Only works with the new release:** leave it on `dev` until the release is
  tagged. Open a PR from `main` with the new pin and the cherry-picked fix;
  have it supersede the Renovate PR and pass the same required checks.
- **The release doesn't need it:** leave the fix on `dev`. Don't merge the
  entire branch or infer which fixes to ship from build history.

Bring relevant changes from `main` back to `dev` as needed so the canary still
exercises the shipping code. Preserve `source-branch: master` when doing so.
No branch-sync bot, green-build tags, or integration App is needed.

## Rollout

1. Start with the existing release, PR, and weekly workflow files and shared
   test action. Confirm Store access for the intended snap and arm64 channels.
   Restrict Store credentials to trusted publishing steps, and serialize
   publishing runs per track so overlapping runs cannot promote out of order.
   Do not promote stale reruns.
2. Set up `dev` and exercise the weekly path with a manual dispatch. Confirm
   it reports the two intended commits and a failed run is visible to maintainers.
3. Prove the candidate test and exact-revision promotion with a supervised
   release. Record the previous stable revision so it can be re-released if
   needed. Enable repository auto-merge and Renovate auto-merge for the pin
   rule only after the required PR check and promotion gate are in place.

### Repository settings and first run

- Open the rollout PR normally. Once its first `Snap PR build and test (arm64)` check appears,
  require that check from **GitHub Actions** in the `main` branch protection
  rule before merging. Keep the PR requirement and force-push protection.
  Protect `dev` similarly after creating it. Leave Actions workflow permissions
  read-only by default; only the checkout-free scheduler requests `actions: write`
  to dispatch the canary.
- Confirm the Renovate GitHub App has repository access. Under **Settings →
  General → Pull Requests**, enable **Allow auto-merge** after the gate is
  working. If branch protection requires one approving review, Renovate must
  wait for that review too. For unattended updates, set required approvals to
  zero while keeping PRs and the build check required, if repository policy
  permits. This changes review requirements for *all* PRs into `main`.
- Under **Settings → Secrets and variables → Actions**, verify `STORE_LOGIN`
  can upload this snap and release arm64 revisions to the four named channels.
  Store tests use ordinary `snap install`, so the runner must be able to
  install the snap without a publisher login.
- Leave the repository **variable** `SNAP_PROMOTE_STABLE` unset during rollout.
  The first merge can publish to candidate, **not stable**. After a successful
  candidate run for the intended release, record the current stable revision
  (`snapcraft status matter-pi-gpio-commander --arch arm64`). To promote that
  first tested revision without rebuilding, run
  `snapcraft release matter-pi-gpio-commander <tested-revision> latest/stable`.
  Set `SNAP_PROMOTE_STABLE=true` for subsequent automatic promotions. To roll
  back, unset it, stop any in-flight stable promotion, and use the same release
  command with the previous known-good revision.
- After `dev` exists, dispatch **Snap release and canary** with **Use workflow
  from: dev**. Use a new dispatch rather than retrying an old weekly
  run. Enable personal GitHub Actions failure notifications for whoever monitors
  the canary. No schedule or Store publication is triggered by a push to `dev`.
