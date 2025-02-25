package e2e

import (
	"context"
	"fmt"
	test_utils "github.com/kluctl/kluctl/v2/e2e/test-utils"
	"github.com/kluctl/kluctl/v2/e2e/test_project"
	"github.com/kluctl/kluctl/v2/pkg/utils/uo"
	"github.com/kluctl/kluctl/v2/pkg/yaml"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"
)

func buildDeployment(name string, namespace string, ready bool) *uo.UnstructuredObject {
	deployment := uo.FromStringMust(fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: nginx
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
`, name, namespace))
	if ready {
		deployment.Merge(uo.FromStringMust(`
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: "2023-03-29T19:23:12Z"
    lastUpdateTime: "2023-03-29T19:23:12Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  - lastTransitionTime: "2023-03-29T19:22:30Z"
    lastUpdateTime: "2023-03-29T19:23:12Z"
    message: ReplicaSet "argocd-redis-8f7689686" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  observedGeneration: 1
  readyReplicas: 1
  replicas: 1
`))
	}
	return deployment
}

func prepareValidateTest(t *testing.T, k *test_utils.EnvTestCluster) *test_project.TestProject {
	p := test_project.NewTestProject(t)

	createNamespace(t, k, p.TestSlug())

	p.UpdateTarget("test", nil)

	p.AddKustomizeDeployment("d1", []test_project.KustomizeResource{
		{Name: fmt.Sprintf("configmap-%s.yml", "d1"), Content: buildDeployment("d1", p.TestSlug(), false)},
	}, nil)

	return p
}

func assertValidate(t *testing.T, p *test_project.TestProject, succeed bool) (string, string) {
	args := []string{"validate"}
	args = append(args, "-t", "test")

	stdout, stderr, err := p.Kluctl(t, args...)

	if succeed {
		assert.NoError(t, err)
		assert.NotContains(t, stdout, fmt.Sprintf("%s/Deployment/d1: readyReplicas field not in status or empty", p.TestSlug()))
		assert.NotContains(t, stderr, "Validation failed")
	} else {
		assert.ErrorContains(t, err, "Validation failed")
		assert.Contains(t, stdout, fmt.Sprintf("%s/Deployment/d1: readyReplicas field not in status or empty", p.TestSlug()))
	}

	return stdout, stderr
}

func TestValidate(t *testing.T) {
	t.Parallel()

	k := defaultCluster1

	p := prepareValidateTest(t, k)

	p.KluctlMust(t, "deploy", "--yes", "-t", "test")
	assertObjectExists(t, k, appsv1.SchemeGroupVersion.WithResource("deployments"), p.TestSlug(), "d1")

	assertValidate(t, p, false)

	readyDeployment := buildDeployment("d1", p.TestSlug(), true)

	_, err := k.DynamicClient.Resource(appsv1.SchemeGroupVersion.WithResource("deployments")).Namespace(p.TestSlug()).
		Patch(context.Background(), "d1", types.ApplyPatchType, []byte(yaml.WriteJsonStringMust(readyDeployment)), metav1.PatchOptions{
			FieldManager: "test",
		}, "status")
	assert.NoError(t, err)

	assertValidate(t, p, true)
}
