# Security Release Process

Emissary-ingress is a large, growing community comprising maintainers, volunteers, and users.
The Emissary community has adopted this security disclosure and response policy to ensure we
responsibly handle critical issues.

This disclosure process draws heavily from that of the Envoy Proxy -- many thanks!

## Emissary Security Team (EMST)

Security vulnerabilities should be handled quickly and sometimes privately. The primary goal of this
process is to reduce the total time users are vulnerable to publicly known exploits.

The Emissary Security Team (EMST) is responsible for organizing the entire response including internal
communication and external disclosure but will need help from relevant developers to successfully
run this process.

The initial Emissary Security Team will consist of all [maintainers](MAINTAINERS.md), with communications
initially handled via email to [secalert@datawire.io](mailto:secalert@datawire.io). In the future,
we may change the membership of the EMST, or the communication mechanism.

## Private Disclosure Process

The Emissary community asks that all suspected vulnerabilities be privately and responsibly disclosed
via email to [secalert@datawire.io](mailto:secalert@datawire.io).

## Public Disclosure Processes

If you know of a publicly disclosed security vulnerability please IMMEDIATELY email
[secalert@datawire.io](mailto:secalert@datawire.io) to inform the Emissary Security Team (EMST)
about the vulnerability so they may start the patch, release, and communication process.

If possible the EMST will ask the person making the public report if the issue can be handled via a
private disclosure process (for example if the full exploit details have not yet been published). If
the reporter denies the request for private disclosure, the EMST will move swiftly with the fix and
release process. In extreme cases GitHub can be asked to delete the issue but this generally isn't
necessary and is unlikely to make a public disclosure less damaging.

## Patch, Release, and Public Communication

For each vulnerability a member of the Emissary Security Team (EMST) will volunteer to lead
coordination with the "Fix Team" and is responsible for sending disclosure emails to the rest of
the community. This lead will be referred to as the "Fix Lead."

The role of Fix Lead should rotate round-robin across the EMST.

Note that, at present, it is likely that the Fix Team and the EMST are identical (all maintainers).
The EMST may decide to bring in additional contributors for added expertise depending on the area
of the code that contains the vulnerability.

All of the timelines below are suggestions and assume a private disclosure. The Fix Lead drives the
schedule using their best judgment based on severity and development time. If the Fix Lead is
dealing with a public disclosure all timelines become ASAP (assuming the vulnerability has a CVSS
score >= 4; see below). If the fix relies on another upstream project's disclosure timeline, that
will adjust the process as well. We will work with the upstream project to fit their timeline and
best protect our users.

### Released versions and the `master` branch

If the vulnerability affects a supported version (typically the _most recent_ minor release, e.g.
1.13), then the full security release process described in this document will be activated. A 
patch release will be created (e.g. 1.13.10) with the fix, and the fix will also be made on 
`master`.

If a vulnerability affects only `master`, the fix will be incorporated into the next release.
Security vulnerabilities that warrant action per this process will be considered release
blockers.

If a security vulnerability affects only unsupported versions but not `master` or a supported
version, no new release will be created. The vulnerability will be described as a GitHub issue,
and a CVE will be filed if warranted by severity.

### Confidentiality, integrity and availability

We consider vulnerabilities leading to the compromise of data confidentiality or integrity to be
our highest priority concerns. Availability, in particular in areas relating to DoS and resource
exhaustion, is also a serious security concern for Emissary operators, given Emissary's common
placement at the edge.

In general, we will fix well-known resource consumption issues (e.g. high CPU or memory usage) in
the open, using our normal process for bugfixes. We will activate the security process for
disclosures that appear to present a significantly higher risk profile than simple "usage", for
example:

* A "query of death", where a single client query can crash Emissary entirely.
* Highly asymmetric resource exhaustion attacks, where very little traffic can cause resource
  exhaustion, e.g. that delivered by a single client.

Note that while we generally consider the installation mechanisms provided by the Emissary-ingress
project (our published Helm charts and manifests) "safe", there is no way to guarantee that the
published installation mechanisms will always work in any specific setting. Ultimately, Emissary
operators need to understand the impact of their own configurations, especially in larger 
installations.

### Fix Team Organization

These steps should be completed within the first 24 hours of disclosure.

- The Fix Lead will work quickly to identify relevant engineers from the affected projects and
  packages and CC those engineers into the disclosure thread. These selected developers are the
  Fix Team.
- Work toward the fix will take place in private "security repos". The Fix Lead is responsible
  for arranging for the Fix Team to have access to the private security repos (cf GitHub's
  documentation on [duplicating a repository](https://docs.github.com/en/github/creating-cloning-and-archiving-repositories/creating-a-repository-on-github/duplicating-a-repository).)

### Fix Development Process

These steps should be completed within the 1-7 days of Disclosure.

- The Fix Lead and the Fix Team will create a
  [CVSS](https://www.first.org/cvss/specification-document) using the [CVSS
  Calculator](https://www.first.org/cvss/calculator/3.0). The Fix Lead makes the final call on the
  calculated CVSS; it is better to move quickly than to spend time making the CVSS perfect.
- The Fix Team will work per the usual [Emissary Development Process](DEVELOPING.md), including
  fix branches, PRs, reviews, etc.
- The Fix Team will notify the Fix Lead that work on the fix branch is complete once the fix is 
  present in the relevant release branch(es) in the private security repo.

If the CVSS score is under 4.0 ([a low severity score](https://www.first.org/cvss/specification-document#i5))
the Fix Team can decide to slow the release process down in the face of holidays, developer
bandwidth, etc. These decisions must be discussed on the secalert mailing list.

### Fix Disclosure Process

With the fix development underway, the Fix Lead needs to come up with an overall communication plan
for the wider community. This Disclosure process should begin after the Fix Team has developed a fix
or mitigation so that a realistic timeline can be communicated to users.

**Disclosure of Forthcoming Fix to Users** (Completed within 1-7 days of Disclosure)

- The Fix Lead will announce in `#emissary` and `#general` on the [Emissary Slack](https://a8r.io/slack)
  informing users that a security vulnerability has been disclosed and that a fix will be made
  available at a specific date and time in the future via this list. This time is the Release Date.
- The Fix Lead will include any mitigating steps users can take until a fix is available.

The communication to users should be actionable. They should know when to block time to apply
patches, understand exact mitigation steps, etc.

**Fix Release Day** (Completed within 1-21 days of Disclosure)

- The Fix Lead will PR the fix from the private security repo into [the Emissary repo](https://github.com/emissary-ingress/emissary).
- Maintainers will merge this PR as quickly as possible. Changes shouldn't be made to the commits even
  for a typo in the CHANGELOG as this will change the git SHA of the commits leading to confusion and
  potentially conflicts as the fix is cherry-picked around branches.
- The Fix Lead will request a CVE from [DWF](https://github.com/distributedweaknessfiling/DWF-Documentation)
  and include the CVSS and release details.
- The Fix Lead will announce in `#emissary` and `#general` on the [Emissary Slack](https://a8r.io/slack) 
  stating the new releases, the CVE number, and the relevant merged PRs to get wide distribution and
  user action. As much as possible this message should be actionable and include links on how to apply
  the fix to user's environments; this can include links to external distributor documentation.
- The Fix Lead will remove the Fix Team from the private security repo.

### Retrospective

These steps should be completed 1-3 days after the Release Date. The retrospective process
[should be blameless](https://landing.google.com/sre/book/chapters/postmortem-culture.html).

- The Fix Lead will send a retrospective of the process to [secalert@datawire.io](mailto:secalert@datawire.io)
  and to `#emissary-dev` on the [Emissary Slack](https://a8r.io/slack), giving details on everyone
  involved, the timeline of the process, links to PRs that introduced the issue (if relevant),
  and any critiques of the response and release process.
- Maintainers and Fix Team are also encouraged to send their own feedback on the process to
  [secalert@datawire.io](mailto:secalert@datawire.io), or to discuss it in `#emissary-dev`
  on the [Emissary Slack](https://a8r.io/slack). Honest critique is the only way we will 
  improve as a community.
