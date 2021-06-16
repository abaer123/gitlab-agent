package kube_testing

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	toIgnore = "ignore-me"
)

func TestIgnoreAnnotation(t *testing.T) {
	for i, a := range equalAnnotatedObjs(t) {
		for j, b := range equalAnnotatedObjs(t) {
			t.Run(fmt.Sprintf("%T %d vs %T %d", a, i, b, j), func(t *testing.T) {
				equal := cmp.Equal(a, b, TransformToUnstructured(), IgnoreAnnotation(toIgnore))
				if !equal {
					assert.True(t, equal, cmp.Diff(a, b, TransformToUnstructured(), IgnoreAnnotation(toIgnore)))
				}
			})
		}
	}
}

func TestTransformToUnstructured(t *testing.T) {
	for i, a := range equalObjs(t) {
		for j, b := range equalObjs(t) {
			t.Run(fmt.Sprintf("%T %d vs %T %d", a, i, b, j), func(t *testing.T) {
				equal := cmp.Equal(a, b, TransformToUnstructured())
				if !equal {
					assert.True(t, equal, cmp.Diff(a, b, TransformToUnstructured()))
				}
			})
		}
	}
}

func equalObjs(t *testing.T) []interface{} {
	return []interface{}{
		testMap(),
		*testMap(),

		ToUnstructured(t, testMap()),
		*ToUnstructured(t, testMap()),

		runtime.Object(testMap()),
		runtime.Object(ToUnstructured(t, testMap())),

		runtime.Unstructured(ToUnstructured(t, testMap())),
	}
}

func equalAnnotatedObjs(t *testing.T) []interface{} {
	return []interface{}{
		// ConfigMap with annotation to ignore
		testMapAnnotated(),
		*testMapAnnotated(),

		ToUnstructured(t, testMapAnnotated()),
		*ToUnstructured(t, testMapAnnotated()),

		runtime.Object(testMapAnnotated()),
		runtime.Object(ToUnstructured(t, testMapAnnotated())),

		runtime.Unstructured(ToUnstructured(t, testMapAnnotated())),

		// ConfigMap without annotation to ignore
		testMapEmptyAnnotations(),
		*testMapEmptyAnnotations(),

		ToUnstructured(t, testMapEmptyAnnotations()),
		*ToUnstructured(t, testMapEmptyAnnotations()),

		runtime.Object(testMapEmptyAnnotations()),
		runtime.Object(ToUnstructured(t, testMapEmptyAnnotations())),

		runtime.Unstructured(ToUnstructured(t, testMapEmptyAnnotations())),
	}
}

func testMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "map2",
			Namespace: "test2",
		},
		Data: map[string]string{
			"key2": "value2",
		},
	}
}

func testMapAnnotated() *corev1.ConfigMap {
	m := testMap()
	if m.Annotations == nil {
		m.Annotations = make(map[string]string)
	}
	m.Annotations[toIgnore] = "a"
	m.Annotations["b"] = "x"
	return m
}

func testMapEmptyAnnotations() *corev1.ConfigMap {
	m := testMap()
	if m.Annotations == nil {
		m.Annotations = make(map[string]string)
	}
	// empty "annotations" field is elided when typed object is marshaled into unstructured.
	// Put something else in there (and above too) to keep the field.
	m.Annotations["b"] = "x"
	return m
}
