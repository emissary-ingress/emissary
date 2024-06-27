## Description

A few sentences describing the overall goals of the pull request's commits.

## Related Issues

List related issues.

## Testing

A few sentences describing what testing you've done, e.g., manual tests, automated tests, deployed in production, etc.

## Checklist

<!--
  Please review the requirements for each checkbox, and check them
  off (change "[ ]" to "[x]") as you verify that they are complete.
-->
- [ ] **Does my change need to be backported to a previous release?**
  - What backport versions were discussed with the Maintainers in the Issue?

- [ ] **I made sure to update `CHANGELOG.md`.**

   Remember, the CHANGELOG needs to mention:
  - Any new features
  - Any changes to our included version of Envoy
  - Any non-backward-compatible changes
  - Any deprecations

- [ ] **This is unlikely to impact how Ambassador performs at scale.**

   Remember, things that might have an impact at scale include:
  - Any significant changes in memory use that might require adjusting the memory limits
  - Any significant changes in CPU use that might require adjusting the CPU limits
  - Anything that might change how many replicas users should use
  - Changes that impact data-plane latency/scalability

- [ ] **My change is adequately tested.**

   Remember when considering testing:
  - Your change needs to be specifically covered by tests.
    - Tests need to cover all the states where your change is relevant: for example, if you add a behavior that can be enabled or disabled, you'll need tests that cover the enabled case and tests that cover the disabled case. It's not sufficient just to test with the behavior enabled.
  - You also need to make sure that the _entire area being changed_ has adequate test coverage.
    - If existing tests don't actually cover the entire area being changed, add tests.
    - This applies even for aspects of the area that you're not changing – check the test coverage, and improve it if needed!
  - We should lean on the bulk of code being covered by unit tests, but...
  - ... an end-to-end test should cover the integration points

- [ ] **I updated `CONTRIBUTING.md` with any special dev tricks I had to use to work on this code efficiently.**

- [ ] **The changes in this PR have been reviewed for security concerns and adherence to security best practices.**
