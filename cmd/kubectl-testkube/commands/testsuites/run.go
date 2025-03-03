package testsuites

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kubeshop/testkube/cmd/kubectl-testkube/commands/common"
	apiv1 "github.com/kubeshop/testkube/pkg/api/v1/client"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

const WatchInterval = 2 * time.Second

func NewRunTestSuiteCmd() *cobra.Command {
	var (
		name                     string
		watchEnabled             bool
		variables                map[string]string
		secretVariables          map[string]string
		executionLabels          map[string]string
		selectors                []string
		concurrencyLevel         int
		httpProxy, httpsProxy    string
		secretVariableReferences map[string]string
		gitBranch                string
		gitCommit                string
		gitPath                  string
		gitWorkingDir            string
		runningContext           string
	)

	cmd := &cobra.Command{
		Use:     "testsuite <testSuiteName>",
		Aliases: []string{"ts"},
		Short:   "Starts new test suite",
		Long:    `Starts new test suite based on TestSuite Custom Resource name, returns results to console`,
		Run: func(cmd *cobra.Command, args []string) {

			startTime := time.Now()
			client, namespace := common.GetClient(cmd)

			var executions []testkube.TestSuiteExecution

			variables, err := common.CreateVariables(cmd, false)
			ui.WarnOnError("getting variables", err)
			options := apiv1.ExecuteTestSuiteOptions{
				ExecutionVariables: variables,
				HTTPProxy:          httpProxy,
				HTTPSProxy:         httpsProxy,
				ExecutionLabels:    executionLabels,
				RunningContext: &testkube.RunningContext{
					Type_:   string(testkube.RunningContextTypeUserCLI),
					Context: runningContext,
				},
			}

			if gitBranch != "" || gitCommit != "" || gitPath != "" || gitWorkingDir != "" {
				options.ContentRequest = &testkube.TestContentRequest{
					Repository: &testkube.RepositoryParameters{
						Branch:     gitBranch,
						Commit:     gitCommit,
						Path:       gitPath,
						WorkingDir: gitWorkingDir,
					},
				}
			}

			switch {
			case len(args) > 0:
				testSuiteName := args[0]
				namespacedName := fmt.Sprintf("%s/%s", namespace, testSuiteName)

				execution, err := client.ExecuteTestSuite(testSuiteName, name, options)
				ui.ExitOnError("starting test suite execution "+namespacedName, err)
				executions = append(executions, execution)
			case len(selectors) != 0:
				selector := strings.Join(selectors, ",")
				executions, err = client.ExecuteTestSuites(selector, concurrencyLevel, options)
				ui.ExitOnError("starting test suite executions "+selector, err)
			default:
				ui.Failf("Pass Test suite name or labels to run by labels ")
			}

			var hasErrors bool
			for _, execution := range executions {
				if execution.IsFailed() {
					hasErrors = true
				}

				if execution.Id != "" {
					if watchEnabled && len(args) > 0 {
						executionCh, err := client.WatchTestSuiteExecution(execution.Id)
						for execution := range executionCh {
							ui.ExitOnError("watching test execution", err)
							printExecution(execution, startTime)
						}
					}

					execution, err = client.GetTestSuiteExecution(execution.Id)
				}

				printExecution(execution, startTime)
				ui.ExitOnError("getting recent execution data id:"+execution.Id, err)

				uiPrintExecutionStatus(execution)

				uiShellTestSuiteGetCommandBlock(execution.Id)
				if execution.Id != "" {
					if !watchEnabled || len(args) == 0 {
						uiShellTestSuiteWatchCommandBlock(execution.Id)
					}
				}
			}

			if hasErrors {
				ui.ExitOnError("executions contain failed on errors")
			}
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "execution name, if empty will be autogenerated")
	cmd.Flags().StringToStringVarP(&variables, "variable", "v", map[string]string{}, "execution variables passed to executor")
	cmd.Flags().StringToStringVarP(&secretVariables, "secret-variable", "s", map[string]string{}, "execution variables passed to executor")
	cmd.Flags().BoolVarP(&watchEnabled, "watch", "f", false, "watch for changes after start")
	cmd.Flags().StringSliceVarP(&selectors, "label", "l", nil, "label key value pair: --label key1=value1")
	cmd.Flags().IntVar(&concurrencyLevel, "concurrency", 10, "concurrency level for multiple test suite execution")
	cmd.Flags().StringVar(&httpProxy, "http-proxy", "", "http proxy for executor containers")
	cmd.Flags().StringVar(&httpsProxy, "https-proxy", "", "https proxy for executor containers")
	cmd.Flags().StringToStringVarP(&executionLabels, "execution-label", "", nil, "execution-label adds a label to execution in form of key value pair: --execution-label key1=value1")
	cmd.Flags().StringToStringVarP(&secretVariableReferences, "secret-variable-reference", "", nil, "secret variable references in a form name1=secret_name1=secret_key1")
	cmd.Flags().StringVarP(&gitBranch, "git-branch", "", "", "if uri is git repository we can set additional branch parameter")
	cmd.Flags().StringVarP(&gitCommit, "git-commit", "", "", "if uri is git repository we can use commit id (sha) parameter")
	cmd.Flags().StringVarP(&gitPath, "git-path", "", "", "if repository is big we need to define additional path to directory/file to checkout partially")
	cmd.Flags().StringVarP(&gitWorkingDir, "git-working-dir", "", "", "if repository contains multiple directories with tests (like monorepo) and one starting directory we can set working directory parameter")
	cmd.Flags().StringVar(&runningContext, "context", "", "running context description for test suite execution")

	return cmd
}
