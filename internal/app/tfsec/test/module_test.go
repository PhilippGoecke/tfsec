package test

import (
	"testing"

	"github.com/aquasecurity/tfsec/internal/app/tfsec/testutil"
	"github.com/stretchr/testify/require"

	"github.com/aquasecurity/defsec/provider"
	"github.com/aquasecurity/defsec/result"
	"github.com/aquasecurity/defsec/rules"
	"github.com/aquasecurity/defsec/severity"
	"github.com/aquasecurity/tfsec/internal/app/tfsec/block"
	"github.com/aquasecurity/tfsec/internal/app/tfsec/parser"
	"github.com/aquasecurity/tfsec/internal/app/tfsec/scanner"
	"github.com/aquasecurity/tfsec/pkg/rule"
)

var badRule = rule.Rule{
	LegacyID: "EXA001",
	DefSecCheck: rules.RuleDef{
		Provider:    provider.AWSProvider,
		Service:     "service",
		ShortCode:   "abc",
		Summary:     "A stupid example check for a test.",
		Impact:      "You will look stupid",
		Resolution:  "Don't do stupid stuff",
		Explanation: "Bad should not be set.",
		Severity:    severity.High,
	},
	BadExample: []string{`
resource "problem" "x" {
bad = "1"
}
`},
	GoodExample: []string{`
resource "problem" "x" {

}
`},
	Links:          nil,
	RequiredTypes:  []string{"resource"},
	RequiredLabels: []string{"problem"},
	CheckTerraform: func(set result.Set, resourceBlock block.Block, _ block.Module) {
		if resourceBlock.GetAttribute("bad").IsTrue() {
			set.AddResult().
				WithDescription("example problem")
		}
	},
}

func Test_ProblemInModuleInSiblingDir(t *testing.T) {

	scanner.RegisterCheckRule(badRule)
	defer scanner.DeregisterCheckRule(badRule)

	fs, err := testutil.NewFilesystem()
	require.NoError(t, err)
	defer fs.Close()

	require.NoError(t, fs.WriteTextFile("/project/main.tf", `
module "something" {
	source = "../modules/problem"
}
`))
	require.NoError(t, fs.WriteTextFile("modules/problem/main.tf", `
resource "problem" "uhoh" {
	bad = true
}
`))

	blocks, err := parser.New(fs.RealPath("/project/"), parser.OptionStopOnHCLError()).ParseDirectory()
	require.NoError(t, err)
	results := scanner.New(scanner.OptionIgnoreCheckErrors(false)).Scan(blocks)
	testutil.AssertCheckCode(t, badRule.ID(), "", results)

}

func Test_ProblemInModuleInSubdirectory(t *testing.T) {

	scanner.RegisterCheckRule(badRule)
	defer scanner.DeregisterCheckRule(badRule)

	fs, err := testutil.NewFilesystem()
	require.NoError(t, err)
	defer fs.Close()

	require.NoError(t, fs.WriteTextFile("project/main.tf", `
module "something" {
	source = "./modules/problem"
}
`))
	require.NoError(t, fs.WriteTextFile("project/modules/problem/main.tf", `
resource "problem" "uhoh" {
	bad = true
}
`))

	blocks, err := parser.New(fs.RealPath("project/"), parser.OptionStopOnHCLError()).ParseDirectory()
	require.NoError(t, err)
	results := scanner.New(scanner.OptionIgnoreCheckErrors(false)).Scan(blocks)
	testutil.AssertCheckCode(t, badRule.ID(), "", results)

}

func Test_ProblemInModuleInParentDir(t *testing.T) {

	scanner.RegisterCheckRule(badRule)
	defer scanner.DeregisterCheckRule(badRule)

	fs, err := testutil.NewFilesystem()
	require.NoError(t, err)
	defer fs.Close()

	require.NoError(t, fs.WriteTextFile("project/main.tf", `
module "something" {
	source = "../problem"
}
`))
	require.NoError(t, fs.WriteTextFile("problem/main.tf", `
resource "problem" "uhoh" {
	bad = true
}
`))

	blocks, err := parser.New(fs.RealPath("project/"), parser.OptionStopOnHCLError()).ParseDirectory()
	require.NoError(t, err)
	results := scanner.New(scanner.OptionIgnoreCheckErrors(false)).Scan(blocks)
	testutil.AssertCheckCode(t, badRule.ID(), "", results)

}

func Test_ProblemInModuleReuse(t *testing.T) {

	scanner.RegisterCheckRule(badRule)
	defer scanner.DeregisterCheckRule(badRule)

	fs, err := testutil.NewFilesystem()
	require.NoError(t, err)
	defer fs.Close()

	require.NoError(t, fs.WriteTextFile("project/main.tf", `
module "something_good" {
	source = "../modules/problem"
	bad = false
}

module "something_bad" {
	source = "../modules/problem"
	bad = true
}
`))
	require.NoError(t, fs.WriteTextFile("modules/problem/main.tf", `
variable "bad" {
	default = false
}
resource "problem" "uhoh" {
	bad = var.bad
}
`))

	blocks, err := parser.New(fs.RealPath("project/"), parser.OptionStopOnHCLError()).ParseDirectory()
	require.NoError(t, err)
	results := scanner.New(scanner.OptionIgnoreCheckErrors(false)).Scan(blocks)
	testutil.AssertCheckCode(t, badRule.ID(), "", results)

}

func Test_ProblemInNestedModule(t *testing.T) {

	scanner.RegisterCheckRule(badRule)
	defer scanner.DeregisterCheckRule(badRule)

	fs, err := testutil.NewFilesystem()
	require.NoError(t, err)
	defer fs.Close()

	require.NoError(t, fs.WriteTextFile("project/main.tf", `
module "something" {
	source = "../modules/a"
}
`))
	require.NoError(t, fs.WriteTextFile("modules/a/main.tf", `
	module "something" {
	source = "../../modules/b"
}
`))
	require.NoError(t, fs.WriteTextFile("modules/b/main.tf", `
module "something" {
	source = "../c"
}
`))
	require.NoError(t, fs.WriteTextFile("modules/c/main.tf", `
resource "problem" "uhoh" {
	bad = true
}
`))

	blocks, err := parser.New(fs.RealPath("project/"), parser.OptionStopOnHCLError()).ParseDirectory()
	require.NoError(t, err)
	results := scanner.New(scanner.OptionIgnoreCheckErrors(false)).Scan(blocks)
	testutil.AssertCheckCode(t, badRule.ID(), "", results)

}

func Test_ProblemInReusedNestedModule(t *testing.T) {

	scanner.RegisterCheckRule(badRule)
	defer scanner.DeregisterCheckRule(badRule)

	fs, err := testutil.NewFilesystem()
	require.NoError(t, err)
	defer fs.Close()

	require.NoError(t, fs.WriteTextFile("project/main.tf", `
module "something" {
  source = "../modules/a"
  bad = false
}

module "something-bad" {
	source = "../modules/a"
	bad = true
}
`))
	require.NoError(t, fs.WriteTextFile("modules/a/main.tf", `
variable "bad" {
	default = false
}
module "something" {
	source = "../../modules/b"
	bad = var.bad
}
`))
	require.NoError(t, fs.WriteTextFile("modules/b/main.tf", `
variable "bad" {
	default = false
}
module "something" {
	source = "../c"
	bad = var.bad
}
`))
	require.NoError(t, fs.WriteTextFile("modules/c/main.tf", `
variable "bad" {
	default = false
}
resource "problem" "uhoh" {
	bad = var.bad
}
`))

	blocks, err := parser.New(fs.RealPath("project/"), parser.OptionStopOnHCLError()).ParseDirectory()
	require.NoError(t, err)
	results := scanner.New(scanner.OptionIgnoreCheckErrors(false)).Scan(blocks)
	testutil.AssertCheckCode(t, badRule.ID(), "", results)

}

func Test_ProblemInInitialisedModule(t *testing.T) {

	scanner.RegisterCheckRule(badRule)
	defer scanner.DeregisterCheckRule(badRule)

	fs, err := testutil.NewFilesystem()
	require.NoError(t, err)
	defer fs.Close()

	require.NoError(t, fs.WriteTextFile("project/main.tf", `
module "something" {
  	source = "/nowhere"
	bad = true
}
`))
	require.NoError(t, fs.WriteTextFile("project/.terraform/modules/a/main.tf", `
variable "bad" {
	default = false
}
resource "problem" "uhoh" {
	bad = var.bad
}
`))
	require.NoError(t, fs.WriteTextFile("project/.terraform/modules/modules.json", `
	{"Modules":[{"Key":"something","Source":"/nowhere","Version":"2.35.0","Dir":".terraform/modules/a"}]}
`))

	blocks, err := parser.New(fs.RealPath("project/"), parser.OptionStopOnHCLError()).ParseDirectory()
	require.NoError(t, err)
	results := scanner.New(scanner.OptionIgnoreCheckErrors(false)).Scan(blocks)
	testutil.AssertCheckCode(t, badRule.ID(), "", results)

}
func Test_ProblemInReusedInitialisedModule(t *testing.T) {

	scanner.RegisterCheckRule(badRule)
	defer scanner.DeregisterCheckRule(badRule)

	fs, err := testutil.NewFilesystem()
	require.NoError(t, err)
	defer fs.Close()

	require.NoError(t, fs.WriteTextFile("project/main.tf", `
module "something" {
  	source = "/nowhere"
	bad = false
} 
module "something2" {
	source = "/nowhere"
  	bad = true
}
`))
	require.NoError(t, fs.WriteTextFile("project/.terraform/modules/a/main.tf", `
variable "bad" {
	default = false
}
resource "problem" "uhoh" {
	bad = var.bad
}
`))
	require.NoError(t, fs.WriteTextFile("project/.terraform/modules/modules.json", `
	{"Modules":[{"Key":"something","Source":"/nowhere","Version":"2.35.0","Dir":".terraform/modules/a"}]}
`))

	blocks, err := parser.New(fs.RealPath("project/"), parser.OptionStopOnHCLError()).ParseDirectory()
	require.NoError(t, err)
	results := scanner.New(scanner.OptionIgnoreCheckErrors(false)).Scan(blocks)
	testutil.AssertCheckCode(t, badRule.ID(), "", results)

}

func Test_ProblemInDuplicateModuleNameAndPath(t *testing.T) {
	scanner.RegisterCheckRule(badRule)
	defer scanner.DeregisterCheckRule(badRule)

	fs, err := testutil.NewFilesystem()
	require.NoError(t, err)
	defer fs.Close()

	require.NoError(t, fs.WriteTextFile("project/main.tf", `
module "something" {
  source = "../modules/a"
  bad = 0
}

module "something-bad" {
	source = "../modules/a"
	bad = 1
}
`))
	require.NoError(t, fs.WriteTextFile("modules/a/main.tf", `
variable "bad" {
	default = 0
}
module "something" {
	source = "../b"
	bad = var.bad
}
`))
	require.NoError(t, fs.WriteTextFile("modules/b/main.tf", `
variable "bad" {
	default = 0
}
module "something" {
	source = "../c"
	bad = var.bad
}
`))
	require.NoError(t, fs.WriteTextFile("modules/c/main.tf", `
variable "bad" {
	default = 0
}
resource "problem" "uhoh" {
	count = var.bad
	bad = true
}
`))

	blocks, err := parser.New(fs.RealPath("project/"), parser.OptionStopOnHCLError()).ParseDirectory()
	require.NoError(t, err)
	results := scanner.New(scanner.OptionIgnoreCheckErrors(false)).Scan(blocks)
	testutil.AssertCheckCode(t, badRule.ID(), "", results)

}

func Test_Dynamic_Variables(t *testing.T) {
	example := `
resource "something" "this" {

	dynamic "blah" {
		for_each = ["a"]

		content {
			ok = true
		}
	}
}
	
resource "bad" "thing" {
	secure = something.this.blah.ok
}
`
	fs, err := testutil.NewFilesystem()
	require.NoError(t, err)
	defer fs.Close()

	require.NoError(t, fs.WriteTextFile("project/main.tf", example))

	r1 := rule.Rule{
		LegacyID: "ABC123",
		DefSecCheck: rules.RuleDef{
			Provider:  provider.AWSProvider,
			Service:   "service",
			ShortCode: "abc123",
			Severity:  severity.High,
		},
		RequiredLabels: []string{"bad"},
		CheckTerraform: func(set result.Set, resourceBlock block.Block, _ block.Module) {
			if resourceBlock.GetAttribute("secure").IsTrue() {
				return
			}
			r := resourceBlock.Range()
			set.AddResult().
				WithDescription("example problem").
				WithRange(r)
		},
	}
	scanner.RegisterCheckRule(r1)
	defer scanner.DeregisterCheckRule(r1)

	blocks, err := parser.New(fs.RealPath("project/"), parser.OptionStopOnHCLError()).ParseDirectory()
	require.NoError(t, err)
	results := scanner.New(scanner.OptionIgnoreCheckErrors(false)).Scan(blocks)
	testutil.AssertCheckCode(t, r1.ID(), "", results)
}

func Test_Dynamic_Variables_FalsePositive(t *testing.T) {
	example := `
resource "something" "else" {
	x = 1
	dynamic "blah" {
		for_each = [true]

		content {
			ok = each.value
		}
	}
}
	
resource "bad" "thing" {
	secure = something.else.blah.ok
}
`
	fs, err := testutil.NewFilesystem()
	require.NoError(t, err)
	defer fs.Close()

	require.NoError(t, fs.WriteTextFile("project/main.tf", example))

	r1 := rule.Rule{
		LegacyID: "ABC123",
		DefSecCheck: rules.RuleDef{
			Provider:  provider.AWSProvider,
			Service:   "service",
			ShortCode: "abc123",
			Severity:  severity.High,
		},
		RequiredLabels: []string{"bad"},
		CheckTerraform: func(set result.Set, resourceBlock block.Block, _ block.Module) {
			if resourceBlock.GetAttribute("secure").IsTrue() {
				return
			}
			r := resourceBlock.Range()
			set.AddResult().
				WithDescription("example problem").
				WithRange(r)
		},
	}
	scanner.RegisterCheckRule(r1)
	defer scanner.DeregisterCheckRule(r1)

	blocks, err := parser.New(fs.RealPath("project/"), parser.OptionStopOnHCLError()).ParseDirectory()
	require.NoError(t, err)
	results := scanner.New(scanner.OptionIgnoreCheckErrors(false)).Scan(blocks)
	testutil.AssertCheckCode(t, "", r1.ID(), results)
}