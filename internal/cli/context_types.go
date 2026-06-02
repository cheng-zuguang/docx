package cli

type configFile struct {
	SchemaVersion        string   `json:"schemaVersion"`
	ContextDir           string   `json:"contextDir"`
	ContextSchemaVersion string   `json:"contextSchemaVersion"`
	EntryFiles           []string `json:"entryFiles"`
	AI                   aiConfig `json:"ai"`
}

type aiConfig struct {
	Provider       string   `json:"provider"`
	Command        string   `json:"command"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
	ContextSources []string `json:"contextSources"`
	Output         string   `json:"output"`
}

type indexFile struct {
	SchemaVersion string                    `json:"schemaVersion"`
	Project       indexProject              `json:"project"`
	ReadOrder     []readOrderEntry          `json:"readOrder"`
	ModuleMap     map[string]moduleMapEntry `json:"moduleMap"`
}

type indexProject struct {
	Name             string   `json:"name"`
	Type             []string `json:"type"`
	PrimaryLanguages []string `json:"primaryLanguages"`
	Summary          string   `json:"summary"`
}

type readOrderEntry struct {
	When    string   `json:"when"`
	Files   []string `json:"files,omitempty"`
	Resolve string   `json:"resolve,omitempty"`
}

type moduleMapEntry struct {
	Paths      []string `json:"paths"`
	Context    string   `json:"context"`
	Confidence string   `json:"confidence"`
}

type projectFile struct {
	SchemaVersion string   `json:"schemaVersion"`
	Name          string   `json:"name"`
	Types         []string `json:"types"`
	Summary       string   `json:"summary"`
	Facts         facts    `json:"facts"`
}

type facts struct {
	Manifests     []string `json:"manifests"`
	LastScannedAt string   `json:"lastScannedAt"`
}

type capabilitiesFile struct {
	SchemaVersion               string               `json:"schemaVersion"`
	AvailableAnalyzers          []analyzerCapability `json:"availableAnalyzers"`
	MissingRecommendedAnalyzers []analyzerCapability `json:"missingRecommendedAnalyzers"`
	LastCheckedAt               string               `json:"lastCheckedAt"`
}

type analyzerCapability struct {
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Languages    []string `json:"languages"`
	Capabilities []string `json:"capabilities"`
	Status       string   `json:"status"`
	InstallHint  string   `json:"installHint,omitempty"`
}

type moduleFile struct {
	SchemaVersion string        `json:"schemaVersion"`
	Module        string        `json:"module"`
	Status        string        `json:"status"`
	Paths         []string      `json:"paths"`
	Summary       moduleSummary `json:"summary"`
	Facts         moduleFacts   `json:"facts"`
	ReadHints     readHints     `json:"readHints"`
	RiskRules     []string      `json:"riskRules"`
	RecentChanges []string      `json:"recentChanges"`
}

type moduleSummary struct {
	Purpose       string   `json:"purpose"`
	OwnedConcepts []string `json:"ownedConcepts"`
	NonGoals      []string `json:"nonGoals"`
}

type moduleFacts struct {
	Entrypoints   []string `json:"entrypoints"`
	PublicAPI     []string `json:"publicApi"`
	Dependencies  []string `json:"dependencies"`
	Dependents    []string `json:"dependents"`
	Tests         []string `json:"tests"`
	LastScannedAt string   `json:"lastScannedAt"`
}

type readHints struct {
	AlwaysRead []string      `json:"alwaysRead"`
	ReadFor    []interface{} `json:"readFor"`
}
