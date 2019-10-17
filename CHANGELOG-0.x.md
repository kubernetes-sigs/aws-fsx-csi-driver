# v0.2.0

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
