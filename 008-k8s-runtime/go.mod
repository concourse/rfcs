module main

go 1.12

replace github.com/concourse/concourse => github.com/concourse/concourse v0.0.0-20190305193436-0cba188c2af6

replace k8s.io/client-go => k8s.io/client-go v2.0.0-alpha.0.0.20190226174127-78295b709ec6+incompatible

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221084156-01f179d85dbc

require (
	github.com/concourse/concourse v0.0.0-00010101000000-000000000000
	github.com/evanphx/json-patch v4.1.0+incompatible // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/knative/pkg v0.0.0-20190402181056-ff46edef0ae5 // indirect
	github.com/mattbaird/jsonpatch v0.0.0-20171005235357-81af80346b1a // indirect
	github.com/tektoncd/pipeline v0.2.0
	golang.org/x/net v0.0.0-20190403144856-b630fd6fe46b // indirect
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/yaml.v2 v2.2.2
	k8s.io/api v0.0.0-20190327184913-92d2ee7fc726
	k8s.io/apimachinery v0.0.0-20190402215020-10bf64c7018d
	k8s.io/client-go v11.0.0+incompatible // indirect
	k8s.io/klog v0.2.0 // indirect
	k8s.io/utils v0.0.0-20190308190857-21c4ce38f2a7 // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)
