package main

import (
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

var platformFileSelectors = map[string]string{
	"hive_aws.go":     "aws",
	"hive_azure.go":   "azure",
	"hive_gcp.go":     "gcp",
	"hive_vsphere.go": "vsphere",
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

func hiveTestsOnly() et.SelectFunction {
	return func(spec *et.ExtensionTestSpec) bool {
		for _, cl := range spec.CodeLocations {
			if strings.Contains(cl, "github.com/openshift/hive") {
				return true
			}
		}
		return false
	}
}

func applyEnvironmentSelectors(specs et.ExtensionTestSpecs) {
	specs.Walk(func(spec *et.ExtensionTestSpec) {
		for _, cl := range spec.CodeLocations {
			for file, platform := range platformFileSelectors {
				if strings.Contains(cl, file) {
					spec.Include(et.PlatformEquals(platform))
					return
				}
			}
		}
	})
}
