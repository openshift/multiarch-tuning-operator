package framework

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
)

func HaveEquivalentNodeAffinity(expected interface{}) types.GomegaMatcher {
	return &equivalentNodeAffinityMatcher{
		expected: expected,
	}
}

type equivalentNodeAffinityMatcher struct {
	expected interface{}
}

func (matcher *equivalentNodeAffinityMatcher) Match(actual interface{}) (success bool, err error) {
	actualPod, ok := actual.(corev1.Pod)
	if !ok {
		return false, fmt.Errorf("HaveEquivalentNodeAffinity matcher expects a *corev1.Pod in the actual value, got %T", actual)
	}
	expectedNodeAffinity, ok := matcher.expected.(*corev1.NodeAffinity)
	if !ok {
		return false, fmt.Errorf("HaveEquivalentNodeAffinity matcher expects a *corev1.NodeAffinity")
	}
	var actualNodeAffinity *corev1.NodeAffinity
	if actualPod.Spec.Affinity != nil && actualPod.Spec.Affinity.NodeAffinity != nil {
		actualNodeAffinity = actualPod.Spec.Affinity.NodeAffinity
	}

	if actualNodeAffinity == nil && expectedNodeAffinity == nil {
		return true, nil
	}
	if actualNodeAffinity == nil || expectedNodeAffinity == nil {
		return false, fmt.Errorf("expectedNodeAffinity: %+v, actualNodeAffinity: %+v", expectedNodeAffinity, actualNodeAffinity)
	}

	if actualNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil && expectedNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		return true, nil
	}
	if actualNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil || expectedNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		return false, fmt.Errorf("expectedNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution: %+v, "+
			"actualNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution: %+v",
			expectedNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
			actualNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
	}

	actualTerms := actualNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	expectedTerms := expectedNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if len(actualTerms) != len(expectedTerms) {
		return false, fmt.Errorf("expectedNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms: %+v, "+
			"actualNodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms: %+v",
			expectedTerms, actualTerms)
	}

	actualTerms = sortMatchExpressions(actualTerms)
	expectedTerms = sortMatchExpressions(expectedTerms)

	// now we can compare with the reflect.DeepEqual method
	return reflect.DeepEqual(actualTerms, expectedTerms), nil
}

func (matcher *equivalentNodeAffinityMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%+v\nto have an equivalent node affinity with \n\t%+v", actual, matcher.expected)
}

func (matcher *equivalentNodeAffinityMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot have an equivalent node affinity with \n\t%#v", actual, matcher.expected)
}

func sortMatchExpressions(nst []corev1.NodeSelectorTerm) []corev1.NodeSelectorTerm {
	for _, term := range nst {
		for _, req := range term.MatchExpressions {
			sort.Strings(req.Values)
		}
		for _, req := range term.MatchFields {
			sort.Strings(req.Values)
		}
		sort.SliceStable(term.MatchExpressions, func(i, j int) bool {
			return term.MatchExpressions[i].Key < term.MatchExpressions[j].Key
		})
		sort.SliceStable(term.MatchFields, func(i, j int) bool {
			return term.MatchFields[i].Key < term.MatchFields[j].Key
		})
	}
	sort.SliceStable(nst, func(i, j int) bool {
		term1 := nst[i]
		term2 := nst[j]
		term1Key := ""
		term2Key := ""
		for _, expr := range term1.MatchExpressions {
			term1Key += expr.Key
		}
		for _, field := range term1.MatchFields {
			term1Key += field.Key
		}
		for _, expr := range term2.MatchExpressions {
			term2Key += expr.Key
		}
		for _, field := range term2.MatchFields {
			term2Key += field.Key
		}
		return term1Key < term2Key
	})
	return nst
}

func HaveEquivalentPreferredNodeAffinity(expected *corev1.NodeAffinity) types.GomegaMatcher {
	return &equivalentPreferredNodeAffinityMatcher{
		expected: expected,
	}
}

type equivalentPreferredNodeAffinityMatcher struct {
	expected interface{}
}

func (matcher *equivalentPreferredNodeAffinityMatcher) Match(actual interface{}) (bool, error) {
	actualPod, ok := actual.(corev1.Pod)
	if !ok {
		return false, fmt.Errorf("HaveEquivalentNodeAffinity matcher expects a *corev1.Pod in the actual value, got %T", actual)
	}
	expectedNodeAffinity, ok := matcher.expected.(*corev1.NodeAffinity)
	if !ok {
		return false, fmt.Errorf("HaveEquivalentNodeAffinity matcher expects a *corev1.NodeAffinity")
	}
	var actualNodeAffinity *corev1.NodeAffinity
	if actualPod.Spec.Affinity != nil && actualPod.Spec.Affinity.NodeAffinity != nil {
		actualNodeAffinity = actualPod.Spec.Affinity.NodeAffinity
	}

	if actualNodeAffinity == nil && expectedNodeAffinity == nil {
		return true, nil
	}
	if actualNodeAffinity == nil || expectedNodeAffinity == nil {
		return false, fmt.Errorf("expectedNodeAffinity: %+v, actualNodeAffinity: %+v", expectedNodeAffinity, actualNodeAffinity)
	}

	if actualNodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution == nil && expectedNodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution == nil {
		return true, nil
	}
	if actualNodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution == nil || expectedNodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution == nil {
		return false, fmt.Errorf("expectedNodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution: %+v, "+
			"actualNodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution: %+v",
			expectedNodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
			actualNodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	}

	actualTerms := actualNodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
	expectedTerms := expectedNodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution

	return matchPreferredNodeAffinity(actualTerms, expectedTerms), nil
}

func (matcher *equivalentPreferredNodeAffinityMatcher) FailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected pod to have equivalent preferred node affinity: %+v", matcher.expected)
}

func (matcher *equivalentPreferredNodeAffinityMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected pod NOT to have equivalent preferred node affinity: %+v", matcher.expected)
}

func matchPreferredNodeAffinity(actualTerms, expectedTerms []corev1.PreferredSchedulingTerm) bool {
	if len(actualTerms) != len(expectedTerms) {
		return false
	}

	// Sort by weight to ensure a consistent comparison order
	sort.SliceStable(actualTerms, func(i, j int) bool {
		return actualTerms[i].Weight < actualTerms[j].Weight
	})
	sort.SliceStable(expectedTerms, func(i, j int) bool {
		return expectedTerms[i].Weight < expectedTerms[j].Weight
	})

	// Compare all preferred scheduling terms
	for i := range actualTerms {
		if actualTerms[i].Weight != expectedTerms[i].Weight {
			return false
		}
		if !reflect.DeepEqual(actualTerms[i].Preference.MatchExpressions, expectedTerms[i].Preference.MatchExpressions) {
			return false
		}
		if !reflect.DeepEqual(actualTerms[i].Preference.MatchFields, expectedTerms[i].Preference.MatchFields) {
			return false
		}
	}

	return true
}
