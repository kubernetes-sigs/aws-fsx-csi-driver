# v0.9.0

### Misc.
* Split e2e into its own module ([#257](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/257), [@olemarkus](https://github.com/olemarkus))
* Added templating for CSIDriver object configuration ([#262](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/262), [@Mberga14](https://github.com/Mberga14))
* Bump klog to klog2 ([#267](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/267), [@olemarkus](https://github.com/olemarkus))
* Bump golang to 1.19 ([#268](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/268), [@olemarkus](https://github.com/olemarkus))
* Check if volume is mounted before unmounting ([#274](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/274), [@khoang98](https://github.com/khoang98))
* upgrade kubernetes dependencies to v0.22.3 ([#276](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/276), [@jacobwolfaws](https://github.com/jacobwolfaws))

# v0.8.3
* Use docker buildx 0.8.x --no-cache-filter to avoid using cached amazonlinux image ([#249](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/249), [@wongma7](https://github.com/wongma7))
* Release 0.8.2 part 3/3 ([#251](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/251), [@amankhunt](https://github.com/amankhunt))
* Use k8s.io/mount-utils instead of k8s.io/utils ([#254](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/254), [@khoang98](https://github.com/khoang98))

# v0.8.2
* Add Idempotent check for mounting node volume ([#246](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/246), [@amankhunt](https://github.com/amankhunt))

# v0.8.1
* Updating to latest AL2 base image.

# v0.8.0

### Misc.
* Release 0.7.0 part 3/3 ([#224](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/224), [@khoang98](https://github.com/khoang98))
* Add make all-push rule ([#228](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/228), [@wongma7](https://github.com/wongma7))
* Release 0.7.1 part 3/3 ([#230](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/230), [@wongma7](https://github.com/wongma7))
* Update ECR sidecars to 1-18-13 ([#231](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/231), [@wongma7](https://github.com/wongma7))

# v0.7.1
* Updating to latest AL2 base image.

# v0.7.0

### New features
* Add ARM support ([#217](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/217), [@khoang98](https://github.com/khoang98))

### Misc.
* Update Doc and Support Template ([#215](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/215), [@CandiedCode](https://github.com/CandiedCode))
* Release 0.6.0 part 3/3: merge previous parts to master ([#216](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/216), [@wongma7](https://github.com/wongma7))
* Bump ginkgo ([#220](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/220), [@wongma7](https://github.com/wongma7))
* Use latest buildx github action and build target platform from build platform  ([#221](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/221), [@wongma7](https://github.com/wongma7))

# v0.6.0

### New features
* Add ExpandVolume (storage scaling) features, tests, and deployment artifacts ([#209](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/209), [@jmwurst](https://github.com/jmwurst))
* Add `WeeklyMaintenanceStartTime` and `FileSystemTypeVersion` as storageclass parameters ([#210](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/210), [@jmwurst](https://github.com/jmwurst))

### Misc.
* Update Dynamic Provisioning S3 README ([#205](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/205), [@wuxingro](https://github.com/wuxingro))

# v0.5.0

### New features
* Add support for Lustre compression ([#186](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/186), [@wuxingro](https://github.com/wuxingro))

### Misc.
* Post-release v0.4.0 ([#166](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/166), [@ayberk](https://github.com/ayberk))
* Fix CI ([#172](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/172), [@wongma7](https://github.com/wongma7))
* Add self to OWNERS ([#173](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/173), [@ayberk](https://github.com/ayberk))
* Update README for stable release ([#177](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/177), [@dimitricole](https://github.com/dimitricole))
* Updated helm chart dns config and imagePullSecrets ([#188](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/188), [@nxf5025](https://github.com/nxf5025))
* go mod tidy && go mod vendor ([#192](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/192), [@wongma7](https://github.com/wongma7))
* Document the stable kustomize overlay not dev ([#193](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/193), [@wongma7](https://github.com/wongma7))
* Helm chart 1.0 ([#194](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/194), [@wongma7](https://github.com/wongma7))

# v0.4.0
[Documentation](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/blob/v0.4.0/docs/README.md)

filename  | sha512 hash
--------- | ------------
[v0.4.0.zip](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/archive/v0.4.0.zip) | `d8d32549cdc9e753b367b32e598f94f214db1c2d7f136b703688292daac670c8bf0aa8165775eba98cfe1beb6db725190989df4d5a38604f8046250d25ee9ef8`
[v0.4.0.tar.gz](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/archive/v0.4.0.tar.gz) | `9f5cc40f074ceb0b5b3cd0fc32ad82728b9866b515c71068fee01732f73ddac502f77491a8f0e7f0bcbcf4ff9539f07333f3f6a18642935518ed9afa3fc6f615`

## Changelog
See [details](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/compare/v0.3.0...v0.4.0) for all the changes.

# v0.3.0
[Documentation](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/blob/v0.3.0/docs/README.md)

filename  | sha512 hash
--------- | ------------
[v0.3.0.zip](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/archive/v0.3.0.zip) | `afe143801536568b159eec7f91f90180d3c645650e578d620ca4b2ccfc4f13548abef1b2db5e963dd98d8623b9e3c2c8dccc31ae0ba04dd8bbfb0be38bca5a65`
[v0.3.0.tar.gz](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/archive/v0.3.0.tar.gz) | `e1e5db7e5e842572ec6276188ae3aeb92f29fcafd80fe1b114e77ee8814f9e3e4c64401d7904a28bf1104a77cf080c3e5a43bce0337be70046b49bb8a7dad9f8`

## Changelog
See [details](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/compare/v0.2.0...v0.3.0) for all the changes.

### Notable changes
* Add support for FSx for lustre create API deployment type ([#130](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/130), @chyz198)
* Update aws sdk for new fsx API and run go mod tidy ([#131](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/131), @wongma7)
* Fix static provisioning example ([#129](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/129), @wongma7)
* Update static provisioning example to include flock mountOption ([#128](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/128), @irlevesque)
* Add conformance tests ([#111](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/111), @leakingtapan)
* Fix paths to dynamic_provisioning_s3 manifests ([#119](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/119), @nathanweeks)
* Add iam:CreateServiceLinkedRole for fsx.amazonaws.com service ([#123](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/123), @leakingtapan)
* Switch to use kustomize ([#122](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/122), @leakingtapan)
* Scope down recommended IAM policy ([#121](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/121), @leakingtapan)
* Fix golangci-lint to 1.21.0 ([#114](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/114), @leakingtapan)

# v0.2.0
[Documentation](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/blob/v0.2.0/docs/README.md)

filename  | sha512 hash
--------- | ------------
[v0.2.0.zip](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/archive/v0.2.0.zip) | `0d0334ec09fa40d30ebe10e9eddd954920d2a15a6c494ccb67b180a9a727c3feaa39728335d473adc7ee8ffdd441c73f6d7498ac3c493f91dfceee58ef57694f`
[v0.2.0.tar.gz](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/archive/v0.2.0.tar.gz) | `16ad40f9b098ee57dfc043990204ac97c90f7b82a5b70eba23e22173d2a9f3c84041498657b29d2db254c4a569e8d6264ecf062197fe4a39a7b38817bab51356`

## Changelog
See [details](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/compare/v0.1.0...v0.2.0) for all the changes.

### Notable changes
* Merge Deployment Manifests ([#51](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/51), [@christopherhein](https://github.com/christopherhein))
* Update README for driver permission and installation ([#52](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/52), [@leakingtapan](https://github.com/leakingtapan))
* Change static and dynamic example to use ReadWriteMany by default ([#58](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/58), [@chyz198](https://github.com/chyz198))
* Update README for org change ([#59](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/59), [@leakingtapan](https://github.com/leakingtapan))
* Update repo references ([#60](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/60), [@chenrui333](https://github.com/chenrui333))
* Add iam policy for FSx driver ([#62](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/62), [@Jeffwan](https://github.com/Jeffwan))
* Update to CSI v1.1.0 ([#69](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/69), [@wongma7](https://github.com/wongma7))
* Added flag for version information output ([#73](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/73), [@wongma7](https://github.com/wongma7))
* Implement mount options support ([#74](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/74), [@wongma7](https://github.com/wongma7))
* Bump driver version to 0.2.0 ([#86](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/86), [@leakingtapan](https://github.com/leakingtapan))
* Bump golang version to 1.12.7 ([#87](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/87), [@leakingtapan](https://github.com/leakingtapan))
* Update CHANGELOG for 0.2.0 ([#90](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/90), [@leakingtapan](https://github.com/leakingtapan))
* Switch to use prow job ([#92](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/92), [@leakingtapan](https://github.com/leakingtapan))
* Update go sdk version for IAM for SA ([#96](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/96), [@leakingtapan](https://github.com/leakingtapan))
* Add support for 1200 GiB and 2400 GiB filesystems ([#98](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/98), [@buzzsurfr](https://github.com/buzzsurfr))
* Update README for IAM policy ([#100](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/100), [@leakingtapan](https://github.com/leakingtapan))
* Update manifest for using EKS IAM for SA ([#102](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/102), [@leakingtapan](https://github.com/leakingtapan))
* Add e2e tests ([#103](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/103), [@leakingtapan](https://github.com/leakingtapan))
* Add e2e test for s3 data repository ([#106](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/106), [@leakingtapan](https://github.com/leakingtapan))

# v0.1.0
[Documentation](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/blob/v0.1.0/docs/README.md)

filename  | sha512 hash
--------- | ------------
[v0.1.0.zip](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/archive/v0.1.0.zip) | `3f6a991028887b58304155820d176ca8f583f98f5c0ec9ba2f72912ff604c0be67ff6bacb818c823c2a87ea9578dfd5cf4db686276e3258aeff6522c55426740`
[v0.1.0.tar.gz](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/archive/v0.1.0.tar.gz) | `2b0ba81ea414ac9ab8f6dc6dbb51648d7830f1ed38a76fe070b7ed6d6d95167b7ee1ef6ab9f8f4b11aedba730921d3f01bb43827c805366b83f3a47f75835d54`

## Changelog

### Notable changes
* Update README for s3 integration example ([#40](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/40), [@leakingtapan](https://github.com/leakingtapan/))
* Support s3 data repository in dynamic provision ([#33](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/33), [@Jeffwan](https://github.com/Jeffwan/))
* Add example for multiple pods ([#22](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/22), [@leakingtapan](https://github.com/leakingtapan/))
* Update README with dynamic provisioning example ([#18](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/18), [@leakingtapan](https://github.com/leakingtapan/))
* Update example for static provisioning ([#17](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/17), [@leakingtapan](https://github.com/leakingtapan/))
* Implement dynamic provisioning for FSx for Lustre PV ([#14](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/14), [@leakingtapan](https://github.com/leakingtapan/))
* Update manifest files ([#11](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/11), [@leakingtapan](https://github.com/leakingtapan/))
* Add sample manifest for multiple pod RWX scenario ([#9](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/9), [@leakingtapan](https://github.com/leakingtapan/))
* Update logging format of the driver ([#4](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/4), [@leakingtapan](https://github.com/leakingtapan/))
* Add travis CI yml ([#2](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/pull/2), [@leakingtapan](https://github.com/leakingtapan/))
* Working version that is CSI 0.3.0 compatible ([30ccc18](https://github.com/kubernetes-sigs/aws-fsx-csi-driver/commit/30ccc18), [@leakingtapan](https://github.com/leakingtapan/))
