# Flaky test quarantine

The default CI test job runs the suite with `-shuffle=on -count=2` to expose
order-dependent and intermittent failures. A test that fails intermittently
must not be hidden with an unconditional skip.

## Quarantine process

1. Open an issue describing the failure, reproduction evidence, and affected
   test.
2. Add one row to the registry below with the issue, owner, date, and planned
   removal condition.
3. Quarantine only the smallest affected test, and leave a comment linking the
   registry entry and issue.
4. Keep the normal suite and the randomized stability job enabled.
5. Remove the registry entry and quarantine when the fix lands.

## Registry

| Test | Issue | Owner | Added | Removal condition |
| --- | --- | --- | --- | --- |
| None | — | — | — | Keep the randomized stability job green |
