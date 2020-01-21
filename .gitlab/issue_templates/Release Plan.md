<!--
Please use the following format for the issue title:

Release Version vX.X.X-gitlab of Container Registry

Example:

Release Version v2.7.3-gitlab of Container Registry
-->

### Milestone

(Please enter the Milestone that this release will target e.g. 12.17)

### Issues

(Please enter the issues that this release should include)

* gitlab-org/container-registry#5 **Example, replace with your own**
* gitlab-org/container-registry#8 **Example, replace with your own**
* gitlab-org/container-registry#13 **Example, replace with your own**

### Release Tasks

These tasks must be completed in order for the release to be considered "In Production."

* [ ]  Follow the release instructions the [Release Instructions](./docs-gitlab/README.md#Releases) in order to Tag a new release. (Do this first!)
* [ ]  Version Bump in [CNG](https://gitlab.com/gitlab-org/build/CNG) Merged: gitlab-org/build/CNG!317 **Example, replace with your own**
* [ ]  Version Bump in [Omnibus-GitLab](https://gitlab.com/gitlab-org/omnibus-gitlab) Merged: gitlab-org/omnibus-gitlab!3862 **Example, replace with your own**
* [ ]  Version Bump in [Charts](https://gitlab.com/gitlab-org/charts) Merged: gitlab-org/charts/gitlab!1105 **Example, replace with your own**
* [ ]  Version Bump in [k8s-workloads](https://gitlab.com/gitlab-com/gl-infra/k8s-workloads/gitlab-com) Merged: gitlab-com/gl-infra/k8s-workloads/gitlab-com!74 **Example, replace with your own**

/label ~"devops::package" ~"group::package" ~"Category:Container Registry" ~backstage ~golang
