### Release the operator (for OCP)

1. Choose a snapshot from the list of snapshots and double check post-submits passed for it.
```shell
oc get snapshots --sort-by .metadata.creationTimestamp -l pac.test.appstudio.openshift.io/event-type=push,appstudio.openshift.io/application=multiarch-tuning-operator
```
2. Look at the results of the tests for the commit reported in the snapshot:
```yaml
[...]
spec:
  application: multiarch-tuning-operator
  artifacts: {}
  components:
  - containerImage: quay.io/redhat-user-workloads/multiarch-tuning-ope-tenant/multiarch-tuning-operator/multiarch-tuning-operator@sha256:250498c137c91f8f932317d48ecacb0d336e705828d3a4163e684933b610547f
    name: multiarch-tuning-operator
    source:
      git:
        revision: d73959c925629f29f9071fd6e7d58a0f58a54399
        url: https://github.com/openshift/multiarch-tuning-operator
  - containerImage: quay.io/redhat-user-workloads/multiarch-tuning-ope-tenant/multiarch-tuning-operator/multiarch-tuning-operator-bundle@sha256:be0945723a0a5ad881d135b3f83a65b0e8fc69c0da337339587aebed4bee89a1
    name: multiarch-tuning-operator-bundle
    source:
      git:
        context: ./
        dockerfileUrl: bundle.Dockerfile
        revision: d73959c925629f29f9071fd6e7d58a0f58a54399
        url: https://github.com/openshift/multiarch-tuning-operator
[...]
```
3. Ensure that the containerImage of the operator is the one referenced by the bundle in the selected snapshot
```shell
podman create quay.io/redhat-user-workloads/multiarch-tuning-ope-tenant/multiarch-tuning-operator/multiarch-tuning-operator-bundle@sha256:...
yq '.spec.install.spec.deployments[0].spec.template.metadata.annotations."multiarch.openshift.io/image"' /tmp/csv.yaml 
```
4. Create a new release for the operator in the Konflux cluster:
```yaml
# oc create -f - <<EOF
apiVersion: appstudio.redhat.com/v1alpha1
kind: Release
metadata:
  generateName: manual-release-
  namespace: multiarch-tuning-ope-tenant
spec:
  releasePlan: multiarch-tuning-operator-release-as-operator
  snapshot: multiarch-tuning-operator-h2rsk
  data:
    releaseNotes:
      type: RHEA
      synopsis: |
        Red Hat Multiarch Tuning 0.9.0
      topic: |
        Red Hat Multiarch Tuning 0.9.0
      description: |
        Red Hat Multiarch Tuning 0.9.0
      solution: |
        Red Hat Multiarch Tuning 0.9.0
      references:
        - https://github.com/openshift/multiarch-tuning-operator
# EOF
```
5. Update the fbc-4.x branches with the SHA of the container image for the bundle and operator, example commit (be aware of the upgrade edges and vertices to implement, TODO: commit example):
  https://github.com/openshift/multiarch-tuning-operator/commit/60cfede0b323e1c900bed5f58f5fff5e4a8891ef
6. Get the new snapshot triggered by the build of the merge commit at the previous point
```shell
oc get snapshots --sort-by .metadata.creationTimestamp -l pac.test.appstudio.openshift.io/event-type=push,appstudio.openshift.io/application=fbc-v4-16  
```
7. Watch the `status` of the `Release` Object or look in the Konflux UI to confirm that images and bundle were published as expected.
8. Create a new Release for the fbc snapshot created after the commit in step 4:
```yaml
# oc create -f - <<EOF
apiVersion: appstudio.redhat.com/v1alpha1
kind: Release
metadata:
  generateName: manual-release-
  namespace: multiarch-tuning-ope-tenant
spec:
  releasePlan: fbc-v4-16-release-as-fbc # fbc-v4-16-release-as-staging-fbc is available for a staging release
  snapshot: fbc-v4-16-49tm9
# EOF
```
9. Repeat the previous 3 steps for each OCP release to change the FBC fragment and operator upgrade graph of.

### Create a new FBC fragment for a new OCP release 

The current approach is that for every new OCP release, we will have to:
1. Create New Konflux Application and Component for the FBC
2. Create a new branch in this repo like `fbc-4.16`. E.g., `fbc-4.17` for OCP 4.17
3. Create a new PR in https://gitlab.cee.redhat.com/releng/konflux-release-data/-/tree/main with ReleasePlanAdmission for the new File Based Catalog (FBC) fragment, based on 
	a. https://gitlab.cee.redhat.com/releng/konflux-release-data/-/blob/main/config/stone-prd-rh01.pg1f.p1/product/ReleasePlanAdmission/multiarch-tuning-ope/multiarch-tuning-operator-fbc-prod-index.yaml
	b. https://gitlab.cee.redhat.com/releng/konflux-release-data/-/blob/main/config/stone-prd-rh01.pg1f.p1/product/ReleasePlanAdmission/multiarch-tuning-ope/multiarch-tuning-operator-fbc-staging-index.yaml
4. Create a new PR in https://github.com/redhat-appstudio/tenants-config to add the ReleasePlan for the new FBC, based on
	a. https://github.com/redhat-appstudio/tenants-config/tree/main/cluster/stone-prd-rh01/tenants/multiarch-tuning-ope-tenant/fbc_4_16/appstudio.redhat.com/releaseplans/fbc-v4-16-release-as-fbc
	b. https://github.com/redhat-appstudio/tenants-config/tree/main/cluster/stone-prd-rh01/tenants/multiarch-tuning-ope-tenant/fbc_4_16/appstudio.redhat.com/releaseplans/fbc-v4-16-release-as-staging-fbc
	c. https://github.com/redhat-appstudio/tenants-config/blob/main/cluster/stone-prd-rh01/tenants/multiarch-tuning-ope-tenant/fbc_4_16/kustomization.yaml
5. Adjust the graphs as needed in the branches.


### Other pre-release SDL activities

If the release is a major version or introduced significant
  [architectural changes](https://docs.engineering.redhat.com/pages/viewpage.action?pageId=402429315),
  ensure that the threat model is reviewed and updated accordingly.

SAST findings will be processed according to:
- https://spaces.redhat.com/display/PRODSEC/SAST+Workflow
- https://spaces.redhat.com/display/PRODSEC/Weakness+Management+Standard
- https://spaces.redhat.com/display/PRODSEC/Security+Errata+Standard

In general, SAST findings cause a PR to fail to merge. 
It's the resposibility of the PR author to review the findings. 

In case a finding is not a false positive, it needs to be classified as weakness or vulnerability 
(if in doubt, contact the Security Architect for help). 
Once classified, an appropriate Jira issue needs to be created (ask Security Architect for guidance). 

The issues will have to be remediated according to the timelines set in the standard.

If malware is detected, contact prodsec@redhat.com immediately.

