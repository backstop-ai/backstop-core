# hermetic/invalid-pack

The pack that must FAIL. It is the negative half of the hermetic remote-dependency
fixture substrate (SPEC-055, TASK-002), consumed by the production validator's
failure claim and by the E2E that proves a remote `pack add` of a bad pack exits
non-zero with the validation diagnostic (CLM-016, CLM-055).

## The named failure

    ./bin/backstop pack check cmd/backstop/testdata/hermetic-remote/invalid-pack
    status: fail
    - phase1-structural: fail
    ERROR [phase1-structural/file-exists] referenced file not found

The rule declares `file: rules/deliberately-absent.yml`, which is not in this
directory. Consuming tests assert on that phase and message, so the failure reason is
part of the fixture contract — not an incidental detail.

## Why THIS failure and not a simpler one

An unparseable manifest, or one missing a required field, would be rejected by the
manifest loader before the validator ever ran; the assertion would still pass while
proving nothing about whether validation is wired. This pack is accepted by the
manifest loader — verified by installing it locally, which succeeds — so the ONLY
thing that rejects it is the pack check pipeline.

Do not create the missing rule file, and do not add claims or fixtures. The pack's
job is to fail, in phase 1, for exactly this reason.
