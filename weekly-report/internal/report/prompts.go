package report

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PromptsConfig holds the LLM system prompt and per-section user templates.
// Edit prompts.yaml to tune the output without recompiling.
type PromptsConfig struct {
	System    string            `yaml:"system"`
	Templates map[string]string `yaml:"templates"`
}

// Template name constants — used as keys into PromptsConfig.Templates.
const (
	tmplExecSummary      = "exec_summary"
	tmplDetailedAnalysis = "detailed_analysis"
	tmplRecommendations  = "recommendations"
	tmplTeamCallout      = "team_callout"
)

// builtinPrompts are used when prompts.yaml is missing or unparseable.
var builtinPrompts = PromptsConfig{
	System: `You are a technical writer who creates concise, insightful reports about
open-source software projects. Your reports are aimed at engineering managers
and technical leads who want a quick overview of project activity.
Write in a professional but approachable tone. Be specific about numbers
and highlight trends. Use markdown formatting for the output.`,

	Templates: map[string]string{
		tmplExecSummary: `Based on the following raw data about activity across the dbt ecosystem
repositories, write an Executive Summary (3-5 paragraphs) that covers:

1. Overall health of the dbt ecosystem this period (activity levels, any
   concerning trends)
2. Key highlights — major PRs merged, important issues filed, new releases
3. Notable patterns — which repos are most active, any repos that need
   attention
4. Community engagement — contributor activity, responsiveness to issues

Keep it concise but insightful. Focus on what a technical leader would
care about.

RAW DATA:
%s`,

		tmplDetailedAnalysis: `Based on the following raw data about activity across the dbt ecosystem
repositories, write a Detailed Analysis section that covers each repository
individually. For each repo include:

1. A brief status summary (1-2 sentences)
2. Key metrics (issues opened/closed ratio, PR merge rate)
3. Notable changes or concerns
4. Any releases and what they mean

After the per-repo analysis, add a "Cross-Repo Trends" subsection noting
any patterns across repositories.

Use markdown formatting with ## headers for each repo.

RAW DATA:
%s`,

		tmplRecommendations: `Based on the following raw data about activity across the dbt ecosystem
repositories, generate a "Recommendations & Action Items" section with:

1. Issues or areas that may need immediate attention
2. Repos that might be under-resourced (high open issue count, low close rate)
3. Suggestions for the coming period

Be specific and actionable. Use bullet points.

RAW DATA:
%s`,

		tmplTeamCallout: `Based on the following data about a specific team's contributions to the dbt
ecosystem, write a Team Callout section. This team is being tracked for
visibility into their ramp-up and contributions.

For each team member, provide:
1. A brief assessment of their activity level and contributions this period
2. What areas/repos they have been working in
3. Notable PRs or issues they have worked on

End with an overall team assessment: how is the team ramping up, any areas
where they could use support, and what is going well.

Be encouraging but factual. If a member had no activity, note it neutrally
(they may have been on other work).

TEAM DATA:
%s`,
	},
}

// LoadPrompts reads a YAML prompts file. Falls back to builtinPrompts on any error.
// Missing keys in the file are filled in from the builtins.
func LoadPrompts(path string) *PromptsConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		return &builtinPrompts
	}
	var p PromptsConfig
	if err := yaml.Unmarshal(data, &p); err != nil {
		fmt.Printf("   warning: could not parse %s: %v — using built-in prompts.\n", path, err)
		return &builtinPrompts
	}
	if p.System == "" {
		p.System = builtinPrompts.System
	}
	if p.Templates == nil {
		p.Templates = builtinPrompts.Templates
	} else {
		// Fill in any missing templates from builtins
		for k, v := range builtinPrompts.Templates {
			if _, ok := p.Templates[k]; !ok {
				p.Templates[k] = v
			}
		}
	}
	return &p
}

// buildPrompt resolves a named template with data substituted into its %s placeholder.
func buildPrompt(p *PromptsConfig, name, data string) (string, error) {
	tmpl, ok := p.Templates[name]
	if !ok {
		return "", fmt.Errorf("unknown prompt template %q", name)
	}
	return fmt.Sprintf(tmpl, data), nil
}
