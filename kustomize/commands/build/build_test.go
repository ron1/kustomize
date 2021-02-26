// Copyright 2019 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"bytes"
	"testing"

	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/konfig"
)

func loadFileSystem(fSys filesys.FileSystem) {
	fSys.WriteFile(konfig.DefaultKustomizationFileName(), []byte(`
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namePrefix: foo-
nameSuffix: -bar
namespace: ns1
commonLabels:
  app: nginx
commonAnnotations:
  note: This is a test annotation
resources:
  - deployment.yaml
  - namespace.yaml
configMapGenerator:
- name: literalConfigMap
  literals:
  - DB_USERNAME=admin
  - DB_PASSWORD=somepw
secretGenerator:
- name: secret
  literals:
    - DB_USERNAME=admin
    - DB_PASSWORD=somepw
  type: Opaque
patchesJson6902:
- target:
    group: apps
    version: v1
    kind: Deployment
    name: dply1
  path: jsonpatch.json
`))
	fSys.WriteFile("deployment.yaml", []byte(`
apiVersion: apps/v1
metadata:
  name: dply1
kind: Deployment
`))
	fSys.WriteFile("namespace.yaml", []byte(`
apiVersion: v1
kind: Namespace
metadata:
  name: ns1
`))
	fSys.WriteFile("jsonpatch.json", []byte(`[
    {"op": "add", "path": "/spec/replica", "value": "3"}
]`))
}

const expectedContent = `apiVersion: v1
kind: Namespace
metadata:
  annotations:
    note: This is a test annotation
  labels:
    app: nginx
  name: ns1
---
apiVersion: v1
data:
  DB_PASSWORD: somepw
  DB_USERNAME: admin
kind: ConfigMap
metadata:
  annotations:
    note: This is a test annotation
  labels:
    app: nginx
  name: foo-literalConfigMap-bar-g5f6t456f5
  namespace: ns1
---
apiVersion: v1
data:
  DB_PASSWORD: c29tZXB3
  DB_USERNAME: YWRtaW4=
kind: Secret
metadata:
  annotations:
    note: This is a test annotation
  labels:
    app: nginx
  name: foo-secret-bar-82c2g5f8f6
  namespace: ns1
type: Opaque
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    note: This is a test annotation
  labels:
    app: nginx
  name: foo-dply1-bar
  namespace: ns1
spec:
  replica: "3"
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      annotations:
        note: This is a test annotation
      labels:
        app: nginx
`

func TestBuild(t *testing.T) {
	fSys := filesys.MakeFsInMemory()
	loadFileSystem(fSys)
	var o Options
	buffy := new(bytes.Buffer)
	cmd := NewCmdBuildWithOptions("build", buffy, &o)
	opts := []string{}
	cmd.Flags().Parse(opts)
	dir := "."
	if err := o.Validate([]string{dir}); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	resMap, err := o.RunBuildWithoutEmitter(fSys)
	if err != nil {
		t.Fatalf("RunBuildWithoutEmitter failed: %v", err)
	}
	o.emitResources(buffy, fSys, resMap)
	if buffy.String() != expectedContent {
		t.Fatalf("Expected output:\n%s\n But got output:\n%s", expectedContent, buffy)
	}
}

func TestNewOptionsToSilenceCodeInspectionError(t *testing.T) {
	if NewOptions("foo", "bar") == nil {
		t.Fatal("could not make new options")
	}
}

func TestBuildValidate(t *testing.T) {
	var cases = []struct {
		name  string
		args  []string
		path  string
		erMsg string
	}{
		{"noargs", []string{}, filesys.SelfDir, ""},
		{"file", []string{"beans"}, "beans", ""},
		{"path", []string{"a/b/c"}, "a/b/c", ""},
		{"path", []string{"too", "many"},
			"",
			"specify one path to " +
				konfig.DefaultKustomizationFileName()},
	}
	for _, mycase := range cases {
		opts := Options{}
		e := opts.Validate(mycase.args)
		if len(mycase.erMsg) > 0 {
			if e == nil {
				t.Errorf("%s: Expected an error %v", mycase.name, mycase.erMsg)
			}
			if e.Error() != mycase.erMsg {
				t.Errorf("%s: Expected error %s, but got %v", mycase.name, mycase.erMsg, e)
			}
			continue
		}
		if e != nil {
			t.Errorf("%s: unknown error: %v", mycase.name, e)
			continue
		}
		if opts.kustomizationPath != mycase.path {
			t.Errorf("%s: expected path '%s', got '%s'", mycase.name, mycase.path, opts.kustomizationPath)
		}
	}
}
