<!--
Please use the following format for the issue title:

Release Version vX.Y.Z-gitlab

Example:

Release Version v2.7.7-gitlab
-->

## Issues
<!--
Please create an unordered list with the issues that this release should include.

Example:

* https://gitlab.com/gitlab-org/gitlab/issues/12345
* https://gitlab.com/gitlab-org/container-registry/issues/12345
-->

## Merge Requests
<!--
(Optional) Please create an unordered list with the merge requests that this release should include.

Example:

* https://gitlab.com/gitlab-org/gitlab/merge_requests/12345
* https://gitlab.com/gitlab-org/container-registry/merge_requests/12345
-->

## Tasks
All tasks must be completed (in order) for the release to be considered ~"workflow::production".

### 1. Prepare

1. [ ] Set the milestone of this issue to the target GitLab release.
1. [ ] Set the due date of this issue to the 12th of the release month.

<details>
<summary><b>Instructions</b></summary>
The due date is set to the 12th of each month to create a buffer of 5 days before the merge deadline on the 17th. See [Product Development Timeline](https://about.gitlab.com/handbook/engineering/workflow/#product-development-timeline) for more information about the GitLab release timings.
</details>

### 2. Release

1. [ ] Create a merge request to add an entry to the [changelog](https://gitlab.com/gitlab-org/container-registry/blob/release/2.8-gitlab/CHANGELOG.md).
1. [ ] Create a [tag](https://gitlab.com/gitlab-org/container-registry/-/tags) for the new release.

<details>
<summary><b>Instructions</b></summary>
Please mention this issue in the description of the changelog merge request.

See [release instructions](https://gitlab.com/gitlab-org/container-registry/tree/release/2.8-gitlab/docs-gitlab#releases) for additional information.
</details>

### 3. Update

1. [ ] Version bump in [CNG](https://gitlab.com/gitlab-org/build/CNG):
    - [ ] Update `GITLAB_CONTAINER_REGISTRY_VERSION` in [`ci_files/variables.yml`](https://gitlab.com/gitlab-org/build/CNG/blob/master/ci_files/variables.yml)
    - [ ] Update `REGISTRY_VERSION` in [`gitlab-container-registry/Dockerfile`](https://gitlab.com/gitlab-org/build/CNG/blob/master/gitlab-container-registry/Dockerfile)
    - [ ] Update `REGISTRY_VERSION` in [`gitlab-container-registry/Dockerfile.build.ubi8`](https://gitlab.com/gitlab-org/build/CNG/blob/master/gitlab-container-registry/Dockerfile.build.ubi8)
    - [ ] Label merge request with: `~/label ~backstage ~"group::distribution" ~"devops::enablement" ~"Workflow::ready for review"`
1. [ ] Version bump in [Omnibus](https://gitlab.com/gitlab-org/omnibus-gitlab):
    - [ ] Create `bump-registry-version-vX-Y-Z-gitlab.yml` in [`changelogs/unreleased`](https://gitlab.com/gitlab-org/omnibus-gitlab/tree/master/changelogs/unreleased)
    - [ ] Update `version` in [`config/software/registry.rb`](https://gitlab.com/gitlab-org/omnibus-gitlab/blob/master/config/software/registry.rb)
    - [ ] Label merge request with: `/label ~backstage ~"group::distribution" ~"devops::enablement" ~"Workflow::ready for review"`
1. [ ] Version bump in [Charts](https://gitlab.com/gitlab-org/charts):
    - [ ] Create `bump-registry-version-vX-Y-Z-gitlab.yml` in [`changelogs/unreleased`](https://gitlab.com/gitlab-org/charts/gitlab/blob/master/changelogs/unreleased)
    - [ ] Update `tag` in [`charts/registry/values.yaml`](https://gitlab.com/gitlab-org/charts/gitlab/blob/master/charts/registry/values.yaml)
    - [ ] Update `tag` in [`doc/charts/registry/index.md`](https://gitlab.com/gitlab-org/charts/gitlab/blob/master/doc/charts/registry/index.md)
    - [ ] Update `version` and `appVersion` in [`charts/registry/Chart.yaml`](https://gitlab.com/gitlab-org/charts/gitlab/-/blob/master/charts/registry/Chart.yaml)
    - [ ] Replace all mentions to the previous `vX.Y.Z-gitlab` release in [`doc/charts/registry/index.md`](https://gitlab.com/gitlab-org/charts/gitlab/blob/master/doc/charts/registry/index.md)
    - [ ] Label merge request with: `/label ~backstage ~"group::distribution" ~"devops::enablement" ~"Workflow::ready for review"`
1. [ ] Version bump in [K8s Workloads](https://gitlab.com/gitlab-com/gl-infra/k8s-workloads/gitlab-com):
    - [ ] Update `$registry_version` in [`init-values.yaml.gotmpl`](https://gitlab.com/gitlab-com/gl-infra/k8s-workloads/gitlab-com/-/blob/master/releases/gitlab/values/init-values.yaml.gotmpl#L75)
    - [ ] Update `tag` in [`gstg.yaml.gotmpl`](https://gitlab.com/gitlab-com/gl-infra/k8s-workloads/gitlab-com/-/blob/master/releases/gitlab/values/gstg.yaml.gotmpl#L6)
    - [ ] Label merge request with: `/label ~"Service::Container Registry" ~"group::distribution" ~"devops::enablement" ~"Workflow::ready for review"`

<details>
<summary><b>Instructions</b></summary>

Bump the Container Registry version used in [CNG](https://gitlab.com/gitlab-org/build/CNG), [Omnibus](https://gitlab.com/gitlab-org/omnibus-gitlab), [Charts](https://gitlab.com/gitlab-org/charts) and [K8s Workloads](https://gitlab.com/gitlab-com/gl-infra/k8s-workloads/gitlab-com).

Create a merge request for each project. Mark parent tasks as completed once the corresponding merge requests are merged.

Version bump merge requests should appear automatically in the `Related merge requests` section of this issue.

Note: According to the [Distribution Team Merge Request Handling](https://about.gitlab.com/handbook/engineering/development/enablement/distribution/merge_requests.html#assigning-merge-requests) documentation, we should not assign merge requests to an individual.

#### Merge Request Template

For consistency, please use the following template for these merge requests:

##### Branch Name

`bump-container-registry-vX-Y-Z-gitlab`

##### Commit Message

`Bump Container Registry to vX.Y.Z-gitlab`

##### Title

`Bump Container Registry to vX.Y.Z-gitlab`

##### Description

Repeat the version subsection for multiple versions. As an example, to bump to v2.7.7 in a project where the current version is v2.7.5, create an entry for v2.7.6 and v2.7.7.

```md
## vX.Y.Z-gitlab
[Changelog](https://gitlab.com/gitlab-org/container-registry/blob/release/X.Y-gitlab/CHANGELOG.md#vXYZ-gitlab-YYYY-MM-DD)

Related to <!-- link to this release issue -->.
```

##### Changelog Entries

Some projects require a changelog entry, please use the following template whenever necessary:

```yml
---
title: Bump Container Registry to vX.Y.Z-gitlab
merge_request: # number (not the link) of the version bump merge request
author:
type: changed
```
</details>

### 4. Complete

1. [ ] Assign label ~"workflow::verification" once all changes have been merged.
1. [ ] Assign label ~"workflow::production" once all changes have been deployed.
1. [ ] Update all related issues, informing that the deploy is complete.
1. [ ] Close this issue.

<details>
<summary><b>Instructions</b></summary>
To see the version deployed in each environment, look at the [Grafana Container Registry dashboard](https://dashboards.gitlab.net/d/registry-pod/registry-pod-info?orgId=1):

![image](/uploads/3fd5b4902472f6cdcc56b9c2d333472f/image.png)
</details>

/label ~"devops::package" ~"group::package" ~"Category:Container Registry" ~golang ~"workflow::scheduling"
