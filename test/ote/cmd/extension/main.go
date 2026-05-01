package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openshift-eng/openshift-tests-extension/pkg/cmd"
	e "github.com/openshift-eng/openshift-tests-extension/pkg/extension"
	et "github.com/openshift-eng/openshift-tests-extension/pkg/extension/extensiontests"
	g "github.com/openshift-eng/openshift-tests-extension/pkg/ginkgo"

	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	// Import test packages to register Ginkgo tests
	_ "github.com/openshift/hive/test/ote/hive"
)

func init() {
	// The OpenShift Ginkgo fork (TRT-2539) introduced ForwardingOutputInterceptor
	// which dups fd 1 and forwards all captured output back to stdout. This breaks
	// OTE's subprocess JSON protocol — non-JSON text appears before the result,
	// causing json.Unmarshal to fail with "Deserializaion Error".
	flag.Set("ginkgo.output-interceptor-mode", "none")
}

var platformFileSelectors = map[string]string{
	"hive_aws.go":     "aws",
	"hive_azure.go":   "azure",
	"hive_gcp.go":     "gcp",
	"hive_vsphere.go": "vsphere",
}

// longRunningTestIDs contains case IDs of tests that provision real clusters
// and take 30-60+ minutes each. These are skipped when SKIP_LONG_RUNNING_TESTS=true.
var longRunningTestIDs = []string{
	"22379", "22381", "23040", "23167", "23986", "24088", "25145", "25210",
	"25310", "25443", "25447", "27559", "28845", "28867", "32026", "32135",
	"32223", "33642", "33832", "33854", "33872", "34148", "35069", "35297",
	"40825", "41212", "41499", "41777", "43100", "44475", "44946", "46016",
	"49471", "52411", "52415", "54463", "63275", "63862", "68240", "68294",
	"75241", "78024", "79046",
}

func main() {
	registry := e.NewRegistry()

	ext := e.NewExtension("openshift", "optional", "hive")

	ext.AddSuite(e.Suite{
		Name:    "openshift/hive",
		Parents: []string{"openshift/conformance/parallel"},
	})

	specs, err := g.BuildExtensionTestSpecsFromOpenShiftGinkgoSuite(hiveTestsOnly())
	if err != nil {
		panic(fmt.Sprintf("couldn't build extension test specs from ginkgo: %+v", err.Error()))
	}

	specs.AddBeforeAll(func() {
		exutil.WithCleanup(func() {})
		if err := compat_otp.InitTest(false); err != nil {
			panic(fmt.Sprintf("failed to initialize test framework: %v", err))
		}
		e2e.AfterReadingAllFlags(compat_otp.TestContext)
	})

	applyEnvironmentSelectors(specs)

	specs.Walk(func(spec *et.ExtensionTestSpec) {
		spec.Lifecycle = et.LifecycleInforming
	})

	ext.AddSpecs(specs)
	registry.Register(ext)

	root := &cobra.Command{
		Long: "OpenShift Hive OTE Extension",
	}

	root.AddCommand(cmd.DefaultExtensionCommands(registry)...)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// testPlatform returns the platform derived from the test's source file.
// Tests with explicit platform tags in their name (e.g. [Hive/GCP])
// override the file-based platform.
func testPlatform(spec *et.ExtensionTestSpec) string {
	if strings.Contains(spec.Name, "[Hive/GCP]") {
		return "gcp"
	}
	for _, cl := range spec.CodeLocations {
		for file, platform := range platformFileSelectors {
			if strings.Contains(cl, file) {
				return platform
			}
		}
	}
	return ""
}

func hiveTestsOnly() et.SelectFunction {
	skipLongRunning := os.Getenv("SKIP_LONG_RUNNING_TESTS") == "true"
	targetPlatform := os.Getenv("PLATFORM")
	return func(spec *et.ExtensionTestSpec) bool {
		for _, cl := range spec.CodeLocations {
			if strings.Contains(cl, "github.com/openshift/hive") {
				if strings.Contains(spec.Name, "Longduration") {
					return false
				}
				if skipLongRunning && isLongRunningTest(spec.Name) {
					return false
				}
				if targetPlatform != "" {
					p := testPlatform(spec)
					if p != "" && p != targetPlatform {
						return false
					}
				}
				return true
			}
		}
		return false
	}
}

func isLongRunningTest(name string) bool {
	for _, id := range longRunningTestIDs {
		if strings.Contains(name, "-"+id+"-") || strings.HasSuffix(name, "-"+id) {
			return true
		}
	}
	return false
}

func applyEnvironmentSelectors(specs et.ExtensionTestSpecs) {
	specs.Walk(func(spec *et.ExtensionTestSpec) {
		p := testPlatform(spec)
		if p != "" {
			spec.Include(et.PlatformEquals(p))
		}
	})
}
