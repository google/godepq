# godepq

A utility for inspecting go import trees

```
Usage of godepq:
  -all-paths=false: whether to include all paths in the result
  -from="": root package
  -ignore="": regular expression for packages to ignore
  -include="": regular expression for packages to include
    (excluding packages matching -ignore)
  -include-stdlib=false: whether to include go standard library imports
  -include-tests=false: whether to include test imports
  -o="list": {list: print path(s), dot: export dot graph}
  -to="": target package for querying dependency paths
```

## Installation:

```
$ go get github.com/google/godepq
```

## Examples:

List the packages imported:
```
$ godepq -from github.com/google/godepq
Packages:
github.com/google/godepq
github.com/google/godepq/deps
```

Find a path between two packages:
```
$ godepq -from k8s.io/kubernetes/pkg/kubelet -to k8s.io/kubernetes/pkg/master
No path found from "k8s.io/kubernetes/pkg/kubelet" to "k8s.io/kubernetes/pkg/master"

$ godepq -from k8s.io/kubernetes/pkg/kubelet -to k8s.io/kubernetes/pkg/credentialprovider
Packages:
k8s.io/kubernetes/pkg/kubelet
k8s.io/kubernetes/pkg/kubelet/dockershim/remote
k8s.io/kubernetes/pkg/kubelet/dockershim
k8s.io/kubernetes/pkg/kubelet/kuberuntime
k8s.io/kubernetes/pkg/credentialprovider
```

Track down how a test package is being pulled into a production binary:
```
$ godepq -from k8s.io/kubernetes/cmd/hyperkube -to net/http/httptest -all-paths -o dot | dot -Tpng -o httptest.png
```

![example output](example.png)

List imported packages, searching only packages which name starts with "k8s.io/kubernetes":
```
$ godepq -from k8s.io/kubernetes/pkg/kubelet -include="^k8s.io/kubernetes" -show-loc
Packages:
k8s.io/kubernetes/pkg/kubelet (6908)
k8s.io/kubernetes/pkg/kubelet/token (175)
k8s.io/kubernetes/pkg/util/removeall (108)
k8s.io/kubernetes/pkg/kubelet/nodestatus (764)
...
...
Total Lines Of Code: 133943
```

*Note: This is not an official Google product.*
