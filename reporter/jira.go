package reporter

import (
	"fmt"
	jira "github.com/andygrunwald/go-jira"
	"github.com/cloudflare/complainer"
	"github.com/cloudflare/complainer/flags"
	"io/ioutil"
	"strings"
)

// jiraReporter holds necessary information to create issues for any failures
type jiraReporter struct {
	client           *jira.Client
	metaProject      *jira.MetaProject
	metaIssuetype    *jira.MetaIssueType
	fieldsConfig     map[string]string
	closedStatusName string
}

func init() {
	var (
		jiraURL             *string
		username            *string
		password            *string
		fieldsConfiguration *string
		closedStatus        *string
	)

	registerMaker("jira", Maker{
		RegisterFlags: func() {
			jiraURL = flags.String("jira.url", "JIRA_URL", "", "default jira instance url")
			username = flags.String("jira.username", "JIRA_USERNAME", "", "User to authenticate to jira as")
			password = flags.String("jira.password", "JIRA_PASSWORD", "", "Password for the user to authenticate")
			fieldsConfiguration = flags.String("jira.fields", "JIRA_FIELDS", "", "all required fields seen in the online form in key:value format seperated by ; for multiple fields. Example - Project:ENG;Issue Type:Bug;Priority:Medium. This configuration MUST contain 'Project', 'Summary', 'Issue Type'")
			closedStatus = flags.String("jira.issue_closed_status", "JIRA_ISSUE_CLOSED_STATUS", "", "The status of issue when it is considered closed. This is used to decide whether or not to create new ticket for the same job")
		},

		Make: func() (Reporter, error) {
			return newJiraReporter(*jiraURL, *username, *password, *fieldsConfiguration, *closedStatus)
		},
	})
}

func newJiraReporter(jiraURL, username, password, fieldsConfiguration, closedStatus string) (*jiraReporter, error) {
	err := checkArgsNotNil(jiraURL, username, password, fieldsConfiguration, closedStatus)
	if err != nil {
		return nil, err
	}

	reporter := new(jiraReporter)
	client, err := createJiraClient(jiraURL, username, password)
	if err != nil {
		return nil, err
	}

	reporter.client = client

	err = reporter.setFieldsConfig(fieldsConfiguration)
	if err != nil {
		return nil, err
	}

	project, found := reporter.fieldsConfig["Project"]
	if !found {
		return nil, fmt.Errorf("project is equired in field configuration")
	}

	issueType, found := reporter.fieldsConfig["Issue Type"]
	if !found {
		return nil, fmt.Errorf("issue type is required in field configuration")
	}

	reporter.closedStatusName = closedStatus

	// get create meta information
	metaProject, err := createMetaProject(client, project)
	if err != nil {
		return nil, err
	}

	reporter.metaProject = metaProject

	// get right issue within project
	metaIssuetype, err := createMetaIssueType(metaProject, issueType)
	if err != nil {
		return nil, err
	}

	// check if the given fields completes the mandatory fields and all listed fields are available
	complete, err := metaIssuetype.CheckCompleteAndAvailable(reporter.fieldsConfig)
	if !complete {
		return nil, err
	}

	reporter.metaIssuetype = metaIssuetype
	return reporter, nil
}

func (j *jiraReporter) Report(failure complainer.Failure, config ConfigProvider, stdoutURL, stderrURL string) error {
	renderedFields := make(map[string]string)
	// render all values as they can be tempaltes
	for field, templatedValue := range j.fieldsConfig {
		rendered, err := fillTemplate(failure, config, stdoutURL, stderrURL, templatedValue)
		if err != nil {
			return fmt.Errorf("rendering value of %s as tempalte failed: %s", field, err)
		}
		renderedFields[field] = rendered
	}

	// generate jql with exact match for summary, project and status
	query := fmt.Sprintf(`summary ~ "\"%s\"" AND project = %s AND status != %s`, renderedFields["Summary"], renderedFields["Project"], j.closedStatusName)
	results, resp, err := j.client.Issue.Search(query, nil)
	if err != nil {
		return fmt.Errorf(readJiraReponse(resp))
	}

	if len(results) != 0 {
		// there were issues not closed.
		// Don't create a new one
		return nil
	}

	issue, err := jira.InitIssueWithMetaAndFields(j.metaProject, j.metaIssuetype, renderedFields)
	if err != nil {
		return fmt.Errorf("could not initialize issue: %s", err)
	}

	_, resp, err = j.client.Issue.Create(issue)
	if err != nil {
		return fmt.Errorf(readJiraReponse(resp))
	}

	return nil
}

// setFieldsConfig gets the fields string in format key:value;key2:value;...
// Seperate them and create a map.
func (j *jiraReporter) setFieldsConfig(fieldsConfiguration string) error {
	fields := strings.Split(fieldsConfiguration, ";")
	templateConfig := make(map[string]string)
	for _, directive := range fields {
		keyValueArr := strings.Split(directive, ":")
		if len(keyValueArr) != 2 {
			return fmt.Errorf("invalid field configuration: expected in key:value format, not %s", directive)
		}
		templateConfig[keyValueArr[0]] = keyValueArr[1]
	}
	j.fieldsConfig = templateConfig
	return nil
}

func getAllIssueTypeNames(project *jira.MetaProject) []string {
	var foundIssueTypes []string
	for _, m := range project.IssueTypes {
		foundIssueTypes = append(foundIssueTypes, m.Name)
	}
	return foundIssueTypes
}

func checkArgsNotNil(args ...string) error {
	for _, value := range args {
		if value == "" {
			return fmt.Errorf("all fields are necessary. Some of them are unfulfilled")
		}
	}
	return nil
}

func readJiraReponse(resp *jira.Response) string {
	if resp == nil || resp.Body == nil {
		return fmt.Sprintf("nil response or response body")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	rawBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("could not read response body. %s", err)
	}

	return fmt.Sprintf("could not create issue. Detailed information: %s", string(rawBody))
}

func createJiraClient(url, username, password string) (*jira.Client, error) {
	jiraClient, err := jira.NewClient(nil, url)
	if err != nil {
		return nil, fmt.Errorf("could not create client: %s", err)
	}

	res, err := jiraClient.Authentication.AcquireSessionCookie(username, password)
	if err != nil || !res {
		return nil, fmt.Errorf("authentication failed: %s", err)
	}

	if !jiraClient.Authentication.Authenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	return jiraClient, nil
}

func createMetaProject(c *jira.Client, project string) (*jira.MetaProject, error) {
	meta, _, err := c.Issue.GetCreateMeta(project)
	if err != nil {
		return nil, fmt.Errorf("failed to get create meta : %s", err)
	}

	// get right project
	metaProject := meta.GetProjectWithKey(project)
	if metaProject == nil {
		return nil, fmt.Errorf("could not find project with key %s", project)
	}

	return metaProject, nil
}

func createMetaIssueType(metaProject *jira.MetaProject, issueType string) (*jira.MetaIssueType, error) {
	metaIssuetype := metaProject.GetIssueTypeWithName(issueType)
	if metaIssuetype == nil {
		return nil, fmt.Errorf("could not find issuetype %s, available are %#v", issueType, getAllIssueTypeNames(metaProject))
	}

	return metaIssuetype, nil
}
