package cmdutil

import (
	"testing"
)

func TestPodFromChoice(t *testing.T) {
	pods := []PodItem{
		{Name: "pod-a", Namespace: "ns1", Display: "pod-a  1/1  Running"},
		{Name: "pod-b", Namespace: "ns2", Display: "pod-b  0/1  Pending"},
	}

	tests := []struct {
		name    string
		choice  string
		wantName string
		wantOk  bool
	}{
		{"match first", "pod-a  1/1  Running", "pod-a", true},
		{"match second", "pod-b  0/1  Pending", "pod-b", true},
		{"no match", "pod-c  1/1  Running", "", false},
		{"empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := podFromChoice(pods, tt.choice)
			if ok != tt.wantOk {
				t.Errorf("podFromChoice(pods, %q) ok = %v, want %v", tt.choice, ok, tt.wantOk)
			}
			if ok && got.Name != tt.wantName {
				t.Errorf("podFromChoice(pods, %q).Name = %q, want %q", tt.choice, got.Name, tt.wantName)
			}
		})
	}
}

func TestSplitPodNamespaceArg(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantNs  string
		wantPod string
		wantOk  bool
	}{
		{"valid", "kube-system/coredns-abc", "kube-system", "coredns-abc", true},
		{"no slash", "coredns-abc", "", "", false},
		{"empty namespace", "/coredns-abc", "", "", false},
		{"empty pod", "kube-system/", "", "", false},
		{"empty", "", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, pod, ok := splitPodNamespaceArg(tt.value)
			if ok != tt.wantOk {
				t.Errorf("splitPodNamespaceArg(%q) ok = %v, want %v", tt.value, ok, tt.wantOk)
			}
			if ns != tt.wantNs || pod != tt.wantPod {
				t.Errorf("splitPodNamespaceArg(%q) = (%q, %q), want (%q, %q)", tt.value, ns, pod, tt.wantNs, tt.wantPod)
			}
		})
	}
}

func TestPodExistsInNamespace(t *testing.T) {
	pods := []PodItem{
		{Name: "pod-a", Namespace: "ns1"},
		{Name: "pod-b", Namespace: "ns2"},
	}

	if !podExistsInNamespace(pods, "ns1", "pod-a") {
		t.Error("expected to find pod-a in ns1")
	}
	if podExistsInNamespace(pods, "ns2", "pod-a") {
		t.Error("expected not to find pod-a in ns2")
	}
	if podExistsInNamespace(pods, "ns1", "pod-c") {
		t.Error("expected not to find pod-c")
	}
}

func TestBuildPodHeader(t *testing.T) {
	tests := []struct {
		name          string
		context       string
		namespace     string
		selector      string
		podQuery      string
		allNamespaces bool
		want          string
	}{
		{"all fields", "ctx", "ns", "app=api", "pod: my-pod", false, "context: ctx  namespace: ns  selector: app=api  pod: my-pod"},
		{"context and namespace only", "ctx", "ns", "", "", false, "context: ctx  namespace: ns"},
		{"namespace only", "", "ns", "", "", false, "namespace: ns"},
		{"empty", "", "", "", "", false, ""},
		{"selector only", "", "", "app=web", "", false, "selector: app=web"},
		{"context and pod query", "ctx", "", "", "pod: api", false, "context: ctx  pod: api"},
		{"all namespaces", "ctx", "", "", "", true, "context: ctx  namespace: all"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildPodHeader(tt.context, tt.namespace, tt.selector, tt.podQuery, tt.allNamespaces)
			if got != tt.want {
				t.Errorf("buildPodHeader(%q, %q, %q, %q, %v) = %q, want %q",
					tt.context, tt.namespace, tt.selector, tt.podQuery, tt.allNamespaces, got, tt.want)
			}
		})
	}
}

func TestFilterPodsByQuery(t *testing.T) {
	pods := []PodItem{
		{Name: "api-server-abc"},
		{Name: "api-server-def"},
		{Name: "web-frontend-123"},
		{Name: "worker-456"},
	}

	tests := []struct {
		name  string
		query string
		want  int
	}{
		{"matches two", "api-server", 2},
		{"matches one", "frontend", 1},
		{"matches none", "database", 0},
		{"matches all with empty substring", "er", 3}, // api-server x2, worker
		{"exact name", "worker-456", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterPodsByQuery(pods, tt.query)
			if len(got) != tt.want {
				t.Errorf("filterPodsByQuery(pods, %q) returned %d results, want %d", tt.query, len(got), tt.want)
			}
		})
	}
}

func TestFilterByQuery(t *testing.T) {
	items := []string{"prod-cluster", "staging-cluster", "dev-cluster"}

	tests := []struct {
		name  string
		query string
		want  int
	}{
		{"matches all", "cluster", 3},
		{"matches one", "staging", 1},
		{"matches none", "test", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterByQuery(items, tt.query)
			if len(got) != tt.want {
				t.Errorf("filterByQuery(items, %q) returned %d results, want %d", tt.query, len(got), tt.want)
			}
		})
	}
}

func TestPodExists(t *testing.T) {
	pods := []PodItem{
		{Name: "pod-a"},
		{Name: "pod-b"},
	}

	if !podExists(pods, "pod-a") {
		t.Error("expected podExists to find pod-a")
	}
	if podExists(pods, "pod-c") {
		t.Error("expected podExists not to find pod-c")
	}
	if podExists(nil, "pod-a") {
		t.Error("expected podExists to return false for nil slice")
	}
}

func TestPodDisplays(t *testing.T) {
	pods := []PodItem{
		{Name: "pod-a", Display: "pod-a  1/1  Running"},
		{Name: "pod-b", Display: ""},
		{Name: "pod-c", Display: "pod-c  0/1  Pending"},
	}

	got := podDisplays(pods)
	expected := []string{"pod-a  1/1  Running", "pod-b", "pod-c  0/1  Pending"}
	if len(got) != len(expected) {
		t.Fatalf("podDisplays returned %d items, want %d", len(got), len(expected))
	}
	for i, v := range got {
		if v != expected[i] {
			t.Errorf("podDisplays[%d] = %q, want %q", i, v, expected[i])
		}
	}
}

func TestContains(t *testing.T) {
	items := []string{"a", "b", "c"}
	if !contains(items, "b") {
		t.Error("expected contains to find b")
	}
	if contains(items, "d") {
		t.Error("expected contains not to find d")
	}
	if contains(nil, "a") {
		t.Error("expected contains to return false for nil slice")
	}
}
