package dispatcher

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
	"k8s.io/test-infra/prow/config"
	prowconfig "k8s.io/test-infra/prow/config"
)

var (
	c = Config{
		Default:       "api.ci",
		NonKubernetes: "app.ci",
		Groups: map[ClusterName]Group{
			"api.ci": {
				Paths: []string{
					".*-postsubmits.yaml$",
					".*openshift/release/.*-periodics.yaml$",
					".*-periodics.yaml$",
				},
				PathREs: []*regexp.Regexp{
					regexp.MustCompile(".*-postsubmits.yaml$"),
					regexp.MustCompile(".*openshift/release/.*-periodics.yaml$"),
					regexp.MustCompile(".*-periodics.yaml$"),
				},
				Jobs: []string{
					"pull-ci-openshift-release-master-build01-dry",
					"pull-ci-openshift-release-master-core-dry",
					"pull-ci-openshift-release-master-services-dry",
					"periodic-acme-cert-issuer-for-build01",
				},
			},
			"ci/api-build01-ci-devcluster-openshift-com:6443": {
				Jobs: []string{
					"periodic-build01-upgrade",
					"periodic-ci-image-import-to-build01",
					"pull-ci-openshift-config-master-format",
					"pull-ci-openshift-psap-special-resource-operator-release-4.6-images",
					"pull-ci-openshift-psap-special-resource-operator-release-4.6-unit",
					"pull-ci-openshift-psap-special-resource-operator-release-4.6-verify",
				},
				Paths: []string{".*openshift-priv/.*-presubmits.yaml$"},
				PathREs: []*regexp.Regexp{
					regexp.MustCompile(".*openshift-priv/.*-presubmits.yaml$"),
				},
			},
		},
	}

	configWithBuildFarm = Config{
		Default:       "api.ci",
		NonKubernetes: "app.ci",
		BuildFarm: map[CloudProvider]JobGroups{
			CloudAWS: {
				ClusterBuild01: {},
			},
			CloudGCP: {
				ClusterBuild02: {},
			},
		},
		Groups: map[ClusterName]Group{
			"api.ci": {
				Paths: []string{
					".*-postsubmits.yaml$",
					".*openshift/release/.*-periodics.yaml$",
					".*-periodics.yaml$",
				},
				PathREs: []*regexp.Regexp{
					regexp.MustCompile(".*-postsubmits.yaml$"),
					regexp.MustCompile(".*openshift/release/.*-periodics.yaml$"),
					regexp.MustCompile(".*-periodics.yaml$"),
				},
				Jobs: []string{
					"pull-ci-openshift-release-master-build01-dry",
					"pull-ci-openshift-release-master-core-dry",
					"pull-ci-openshift-release-master-services-dry",
					"periodic-acme-cert-issuer-for-build01",
				},
			},
			"build01": {
				Jobs: []string{
					"periodic-build01-upgrade",
					"periodic-ci-image-import-to-build01",
					"pull-ci-openshift-config-master-format",
					"pull-ci-openshift-psap-special-resource-operator-release-4.6-images",
					"pull-ci-openshift-psap-special-resource-operator-release-4.6-unit",
					"pull-ci-openshift-psap-special-resource-operator-release-4.6-verify",
				},
				Paths: []string{".*openshift-priv/.*-presubmits.yaml$"},
				PathREs: []*regexp.Regexp{
					regexp.MustCompile(".*openshift-priv/.*-presubmits.yaml$"),
				},
			},
		},
	}

	configWithBuildFarmWithJobs = Config{
		Default:       "api.ci",
		NonKubernetes: "app.ci",
		BuildFarm: map[CloudProvider]JobGroups{
			CloudAWS: {
				ClusterBuild01: {
					Paths: []string{
						".*some-build-farm-presubmits.yaml$",
					},
					PathREs: []*regexp.Regexp{
						regexp.MustCompile(".*some-build-farm-presubmits.yaml$"),
					},
				},
			},
			CloudGCP: {
				ClusterBuild02: {},
			},
		},
		Groups: map[ClusterName]Group{
			"api.ci": {
				Paths: []string{
					".*-postsubmits.yaml$",
					".*openshift/release/.*-periodics.yaml$",
					".*-periodics.yaml$",
				},
				PathREs: []*regexp.Regexp{
					regexp.MustCompile(".*-postsubmits.yaml$"),
					regexp.MustCompile(".*openshift/release/.*-periodics.yaml$"),
					regexp.MustCompile(".*-periodics.yaml$"),
				},
				Jobs: []string{
					"pull-ci-openshift-release-master-build01-dry",
					"pull-ci-openshift-release-master-core-dry",
					"pull-ci-openshift-release-master-services-dry",
					"periodic-acme-cert-issuer-for-build01",
				},
			},
			"build01": {
				Jobs: []string{
					"periodic-build01-upgrade",
					"periodic-ci-image-import-to-build01",
					"pull-ci-openshift-config-master-format",
					"pull-ci-openshift-psap-special-resource-operator-release-4.6-images",
					"pull-ci-openshift-psap-special-resource-operator-release-4.6-unit",
					"pull-ci-openshift-psap-special-resource-operator-release-4.6-verify",
				},
				Paths: []string{".*openshift-priv/.*-presubmits.yaml$"},
				PathREs: []*regexp.Regexp{
					regexp.MustCompile(".*openshift-priv/.*-presubmits.yaml$"),
				},
			},
		},
	}
)

func TestLoadConfig(t *testing.T) {
	testCases := []struct {
		name string

		configPath    string
		expected      *Config
		expectedError error
	}{
		{
			name:          "file not exist",
			expectedError: fmt.Errorf("failed to read the config file \"testdata/TestLoadConfig/file_not_exist.yaml\": open testdata/TestLoadConfig/file_not_exist.yaml: no such file or directory"),
		},
		{
			name:          "invalid yaml",
			expectedError: fmt.Errorf("failed to unmarshal the config \"invalid yaml format\\n\": error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go value of type dispatcher.Config"),
		},
		{
			name:          "invalid regex",
			expectedError: fmt.Errorf("[failed to compile regex config.Groups[default].Paths[0] from \"[\": error parsing regexp: missing closing ]: `[`, failed to compile regex config.Groups[default].Paths[1] from \"[0-9]++\": error parsing regexp: invalid nested repetition operator: `++`]"),
		},
		{
			name:     "good config",
			expected: &c,
		},
		{
			name:     "good config with build farm",
			expected: &configWithBuildFarm,
		},
		{
			name:     "good config with build farm with jobs",
			expected: &configWithBuildFarmWithJobs,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := LoadConfig(filepath.Join("testdata", fmt.Sprintf("%s.yaml", t.Name())))
			if !reflect.DeepEqual(tc.expected, actual) {
				t.Errorf("%s: actual differs from expected:\n%s", t.Name(), cmp.Diff(tc.expected, actual))
			}
			equalError(t, tc.expectedError, err)
		})
	}
}

func TestGetClusterForJob(t *testing.T) {
	testCases := []struct {
		name string

		config   *Config
		jobBase  prowconfig.JobBase
		path     string
		expected ClusterName
	}{
		{
			name:     "some job",
			config:   &c,
			jobBase:  config.JobBase{Agent: "kubernetes", Name: "some-job"},
			path:     "org/repo/some-postsubmits.yaml",
			expected: "api.ci",
		},
		{
			name:     "job must on build01",
			config:   &c,
			jobBase:  config.JobBase{Agent: "kubernetes", Name: "periodic-build01-upgrade"},
			expected: "ci/api-build01-ci-devcluster-openshift-com:6443",
		},
		{
			name:     "some periodic job in release repo",
			config:   &c,
			jobBase:  config.JobBase{Agent: "kubernetes", Name: "promote-release-openshift-machine-os-content-e2e-aws-4.1"},
			path:     "ci-operator/jobs/openshift/release/openshift-release-release-4.1-periodics.yaml",
			expected: "api.ci",
		},
		{
			name:     "some jenkins job",
			config:   &c,
			jobBase:  config.JobBase{Agent: "jenkins", Name: "test_branch_wildfly_images"},
			path:     "ci-operator/jobs/openshift-s2i/s2i-wildfly/openshift-s2i-s2i-wildfly-master-postsubmits.yaml",
			expected: "app.ci",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.config.GetClusterForJob(tc.jobBase, tc.path)
			if !reflect.DeepEqual(tc.expected, actual) {
				t.Errorf("%s: actual differs from expected:\n%s", t.Name(), cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestDetermineClusterForJob(t *testing.T) {
	testCases := []struct {
		name string

		config                 *Config
		jobBase                prowconfig.JobBase
		path                   string
		expected               ClusterName
		expectedCanBeRelocated bool
	}{
		{
			name:     "some job",
			config:   &configWithBuildFarmWithJobs,
			jobBase:  config.JobBase{Agent: "kubernetes", Name: "some-job"},
			path:     "org/repo/some-postsubmits.yaml",
			expected: "api.ci",
		},
		{
			name:     "job must on build01",
			config:   &configWithBuildFarmWithJobs,
			jobBase:  config.JobBase{Agent: "kubernetes", Name: "periodic-build01-upgrade"},
			expected: "build01",
		},
		{
			name:     "some periodic job in release repo",
			config:   &configWithBuildFarmWithJobs,
			jobBase:  config.JobBase{Agent: "kubernetes", Name: "promote-release-openshift-machine-os-content-e2e-aws-4.1"},
			path:     "ci-operator/jobs/openshift/release/openshift-release-release-4.1-periodics.yaml",
			expected: "api.ci",
		},
		{
			name:     "some jenkins job",
			config:   &configWithBuildFarmWithJobs,
			jobBase:  config.JobBase{Agent: "jenkins", Name: "test_branch_wildfly_images"},
			path:     "ci-operator/jobs/openshift-s2i/s2i-wildfly/openshift-s2i-s2i-wildfly-master-postsubmits.yaml",
			expected: "app.ci",
		},
		{
			name:                   "some job in build farm",
			config:                 &configWithBuildFarmWithJobs,
			jobBase:                config.JobBase{Agent: "kubernetes", Name: "some-build-farm-job"},
			path:                   "org/repo/some-build-farm-presubmits.yaml",
			expected:               "build01",
			expectedCanBeRelocated: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, canBeRelocated := tc.config.DetermineClusterForJob(tc.jobBase, tc.path)
			if !reflect.DeepEqual(tc.expected, actual) {
				t.Errorf("%s: actual differs from expected:\n%s", t.Name(), cmp.Diff(tc.expected, actual))
			}
			if !reflect.DeepEqual(tc.expectedCanBeRelocated, canBeRelocated) {
				t.Errorf("%s: actual differs from expected:\n%s", t.Name(), cmp.Diff(tc.expectedCanBeRelocated, canBeRelocated))
			}
		})
	}
}

func TestIsInBuildFarm(t *testing.T) {
	testCases := []struct {
		name        string
		config      *Config
		clusterName ClusterName
		expected    CloudProvider
	}{
		{
			name:        "build01",
			config:      &configWithBuildFarm,
			clusterName: ClusterBuild01,
			expected:    "aws",
		},
		{
			name:        "app.ci",
			config:      &configWithBuildFarm,
			clusterName: ClusterAPPCI,
			expected:    "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.config.IsInBuildFarm(tc.clusterName)
			if !reflect.DeepEqual(tc.expected, actual) {
				t.Errorf("%s: actual differs from expected:\n%s", t.Name(), cmp.Diff(tc.expected, actual))
			}
		})
	}
}

func TestMatchingPathRegEx(t *testing.T) {
	testCases := []struct {
		name     string
		config   *Config
		path     string
		expected bool
	}{
		{
			name:     "matching: true",
			config:   &c,
			path:     "./ci-operator/jobs/openshift/ci-tools/openshift-ci-tools-master-postsubmits.yaml",
			expected: true,
		},
		{
			name:   "matching: false",
			config: &c,
			path:   "./ci-operator/jobs/openshift/ci-tools/openshift-ci-tools-master-presubmits.yaml",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.config.MatchingPathRegEx(tc.path)
			if !reflect.DeepEqual(tc.expected, actual) {
				t.Errorf("%s: actual differs from expected:\n%s", t.Name(), cmp.Diff(tc.expected, actual))
			}
		})
	}
}
