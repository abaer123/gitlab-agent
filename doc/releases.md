# Release Process

1. On the 15th, an automatic Slack notification reminds the Configure team to create a monthly release.
1. On the 15th we should always tag a new version that matches the upcoming GitLab minor version. Eg. If GitLab 13.7 will be released on the 22nd, then we should tag 13.7.0.
1. The GITLAB_KAS_VERSION in the GitLab rails monolith is updated to that new tag in a new MR. This MR should be accepted by the maintainer with no questions asked, typically
1. An end-to-end QA test with a real agent and GDK is run automatically as part of the nightly QA process.
1. If there are breaking changes to the kas config file, then MRs need to be raised for Omnibus, and charts.
1. The GITLAB_KAS_VERSION is automatically synced for Omnibus and chart releases.
