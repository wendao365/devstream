package configmanager

import (
	"fmt"

	"github.com/devstream-io/devstream/pkg/util/log"
	"github.com/devstream-io/devstream/pkg/util/scm"
	"github.com/devstream-io/devstream/pkg/util/scm/git"
)

const (
	repoScaffoldingPluginName = "repo-scaffolding"
)

type repoTemplate struct {
	*scm.SCMInfo `yaml:",inline"`
	Vars         RawOptions `yaml:"vars"`
}

type app struct {
	Name         string        `yaml:"name" mapstructure:"name"`
	Spec         *appSpec      `yaml:"spec" mapstructure:"spec"`
	Repo         *scm.SCMInfo  `yaml:"repo" mapstructure:"repo"`
	RepoTemplate *repoTemplate `yaml:"repoTemplate" mapstructure:"repoTemplate"`
	CIRawConfigs []pipelineRaw `yaml:"ci" mapstructure:"ci"`
	CDRawConfigs []pipelineRaw `yaml:"cd" mapstructure:"cd"`

	// these two variables is used internal for convince
	// repoInfo is generated from Repo field with setDefault method
	repoInfo *git.RepoInfo `yaml:"-" mapstructure:"-"`
	// repoTemplateInfo is generated from RepoTemplate field with setDefault method
	repoTemplateInfo *git.RepoInfo `yaml:"-" mapstructure:"-"`
}

func (a *app) getTools(vars map[string]any, templateMap map[string]string) (Tools, error) {
	// 1. set app default field repoInfo and repoTemplateInfo
	if err := a.setDefault(); err != nil {
		return nil, err
	}

	// 2. get ci/cd pipelineTemplates
	appVars := a.Spec.merge(vars)
	tools, err := a.generateCICDTools(templateMap, appVars)
	if err != nil {
		return nil, fmt.Errorf("app[%s] get pipeline tools failed: %w", a.Name, err)
	}

	// 3. generate app repo and template repo from scmInfo
	repoScaffoldingTool := a.generateRepoTemplateTool()
	if repoScaffoldingTool != nil {
		tools = append(tools, repoScaffoldingTool)
	}
	log.Debugf("Have got %d tools from app %s.", len(tools), a.Name)
	return tools, nil
}

// generateAppPipelineTool generate ci/cd tools from app config
func (a *app) generateCICDTools(templateMap map[string]string, appVars map[string]any) (Tools, error) {
	allPipelineRaw := append(a.CIRawConfigs, a.CDRawConfigs...)
	var tools Tools
	// pipelineGlobalVars is used to pass variable from ci/cd pipelines
	pipelineGlobalVars := a.newPipelineGlobalOptionFromApp()
	for _, p := range allPipelineRaw {
		t, err := p.getPipelineTemplate(templateMap, appVars)
		if err != nil {
			return nil, err
		}
		t.updatePipelineVars(pipelineGlobalVars)
		pipelineTool, err := t.generatePipelineTool(pipelineGlobalVars)
		if err != nil {
			return nil, err
		}
		pipelineTool.DependsOn = a.getRepoTemplateDependants()
		tools = append(tools, pipelineTool)
	}
	return tools, nil
}

// generateRepoTemplateTool will use repo-scaffolding plugin for app
func (a *app) generateRepoTemplateTool() *Tool {
	if a.repoTemplateInfo != nil {
		templateVars := make(RawOptions)
		// templateRepo doesn't need auth info
		if a.RepoTemplate.Vars == nil {
			templateVars = make(RawOptions)
		}
		return newTool(
			repoScaffoldingPluginName, a.Name, RawOptions{
				"destinationRepo": RawOptions(a.repoInfo.Encode()),
				"sourceRepo":      RawOptions(a.repoTemplateInfo.Encode()),
				"vars":            templateVars,
			},
		)
	}
	return nil
}

// setDefault will set repoName to appName if repo.name field is empty
func (a *app) setDefault() error {
	if a.Repo == nil {
		return fmt.Errorf("configmanager[app] is invalid, repo field must be configured")
	}
	if a.Repo.Name == "" {
		a.Repo.Name = a.Name
	}
	appRepo, err := a.Repo.BuildRepoInfo()
	if err != nil {
		return fmt.Errorf("configmanager[app] parse repo failed: %w", err)
	}
	a.repoInfo = appRepo
	if a.RepoTemplate != nil {
		// templateRepo doesn't need auth info
		templateRepo, err := a.RepoTemplate.BuildRepoInfo()
		if err != nil {
			return fmt.Errorf("configmanager[app] parse repoTemplate failed: %w", err)
		}
		templateRepo.NeedAuth = false
		a.repoTemplateInfo = templateRepo
	}
	return nil
}

// since all plugin depends on code is deployed, get dependsOn for repoTemplate
func (a *app) getRepoTemplateDependants() []string {
	var dependsOn []string
	// if a.RepoTemplate is configured, pipeline need to wait reposcaffolding finished
	if a.RepoTemplate != nil {
		dependsOn = []string{fmt.Sprintf("%s.%s", repoScaffoldingPluginName, a.Name)}
	}
	return dependsOn
}

// newPipelineGlobalOptionFromApp generate pipeline options used for pipeline option configuration
func (a *app) newPipelineGlobalOptionFromApp() *pipelineGlobalOption {
	return &pipelineGlobalOption{
		RepoInfo: a.repoInfo,
		AppSpec:  a.Spec,
		Scm:      a.Repo,
		AppName:  a.Name,
	}
}
