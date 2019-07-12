package _import

import (
	"fmt"
	"github.com/jenkins-x/bdd-jx/test/helpers"
	"github.com/jenkins-x/bdd-jx/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gopkg.in/src-d/go-git.v4"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var _ = AllImportsTest()

var (
	IncludedImports = os.Getenv("JX_BDD_IMPORTS")
)

// AllImportsTest creates all the tests for all the quickstarts that we want to import
func AllImportsTest() []bool {
	if IncludedImports != "" {
		scenarios := strings.Split(strings.TrimSpace(IncludedImports), ",")
		tests := make([]bool, len(scenarios))
		for _, scenarioName := range scenarios {
			tests = append(tests, createTest(scenarioName, fmt.Sprintf("https://github.com/jenkins-x-quickstarts/%s", scenarioName)))
		}
		return tests
	} else {
		return make([]bool, 0)
	}
}

// createTest creates each test for every scenario we want to test
func createTest(quickstartName string, repoToImport string) bool {
	return Describe("Creating application "+quickstartName, func() {
		var T helpers.TestOptions

		BeforeEach(func() {
			qsNameParts := strings.Split(quickstartName, "-")
			qsAbbr := ""
			for s := range qsNameParts {
				qsAbbr = qsAbbr + qsNameParts[s][:1]

			}
			applicationName := helpers.TempDirPrefix + qsAbbr + "-import-" + strconv.FormatInt(GinkgoRandomSeed(), 10)
			T = helpers.TestOptions{
				ApplicationName: applicationName,
				WorkDir:         helpers.WorkDir,
			}
			T.GitProviderURL()
		})

		Context("by running jx import", func() {
			It("creates an application from the specified folder and promotes it to staging", func() {
				destDir := T.WorkDir + "/" + T.ApplicationName

				By(fmt.Sprintf("calling git clone %s", repoToImport), func() {
					_, err := git.PlainClone(destDir, false, &git.CloneOptions{
						URL:      repoToImport,
						Progress: GinkgoWriter,
					})
					Expect(err).NotTo(HaveOccurred())
				})

				By("removing the .git directory", func() {
					err := os.RemoveAll(destDir + "/.git")
					utils.ExpectNoError(err)
					Expect(destDir + "/.git").ToNot(BeADirectory())
				})

				By("updating the pom.xml (if exists) to have the correct application name", func() {
					_ = utils.ReplaceElement(filepath.Join(destDir, "pom.xml"), "artifactId", T.ApplicationName, 1)
				})

				gitProviderUrl, err := T.GitProviderURL()
				Expect(err).NotTo(HaveOccurred())
				args := []string{"import", destDir, "-b", "--org", T.GetGitOrganisation(), "--git-provider-url", gitProviderUrl}
				argsStr := strings.Join(args, " ")
				By(fmt.Sprintf("running jx %s", argsStr), func() {
					T.ExpectJxExecution(T.WorkDir, helpers.TimeoutSessionWait, 0, args...)
				})

				T.TheApplicationShouldBeBuiltAndPromotedViaCICD(200)

				if T.DeleteApplications() {
					args = []string{"delete", "application", "-b", T.ApplicationName}
					argsStr := strings.Join(args, " ")
					By(fmt.Sprintf("deleting the application by calling jx %s", argsStr), func() {
						T.ExpectJxExecution(T.WorkDir, helpers.TimeoutSessionWait, 0, args...)
					})

				}

				if T.DeleteRepos() {
					args = []string{"delete", "repo", "-b", "-g", gitProviderUrl, "-o", T.GetGitOrganisation(), "-n", T.ApplicationName}
					argsStr := strings.Join(args, " ")
					By(fmt.Sprintf("deleting the repo by calling jx %s", argsStr), func() {
						T.ExpectJxExecution(T.WorkDir, helpers.TimeoutSessionWait, 0, args...)
					})
				}
			})
		})
	})
}
