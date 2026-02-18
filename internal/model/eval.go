package model

// EvalRecord represents a single evaluation run.
type EvalRecord struct {
	ID              string        `json:"id,omitempty"`
	ArtifactName    string        `json:"artifactName"`
	ArtifactVersion string        `json:"artifactVersion"`
	ArtifactKind    Kind          `json:"artifactKind"`
	Category        EvalCategory  `json:"category"`
	Provider        *EvalProvider `json:"provider,omitempty"`
	Benchmark       Benchmark     `json:"benchmark"`
	Evaluator       Evaluator     `json:"evaluator"`
	Results         EvalResults   `json:"results"`
	Context         *EvalContext  `json:"context,omitempty"`
	CreatedAt       string        `json:"createdAt,omitempty"`
}

type EvalCategory string

const (
	EvalCategoryFunctional  EvalCategory = "functional"
	EvalCategorySafety      EvalCategory = "safety"
	EvalCategoryRedTeam     EvalCategory = "red-team"
	EvalCategoryPerformance EvalCategory = "performance"
	EvalCategoryCustom      EvalCategory = "custom"
)

type EvalProvider struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	URL     string `json:"url,omitempty"`
}

type Benchmark struct {
	Name        string `json:"name"`
	Version     string `json:"version,omitempty"`
	Suite       string `json:"suite,omitempty"`
	Description string `json:"description,omitempty"`
}

type Evaluator struct {
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Type     string `json:"type"`
	Identity string `json:"identity,omitempty"`
}

type EvalResults struct {
	OverallScore float64                `json:"overallScore"`
	Metrics      map[string]MetricValue `json:"metrics,omitempty"`
	PerTask      []TaskResult           `json:"perTask,omitempty"`
}

type MetricValue struct {
	Value          float64 `json:"value"`
	Unit           string  `json:"unit,omitempty"`
	HigherIsBetter *bool   `json:"higher_is_better,omitempty"`
}

type TaskResult struct {
	TaskID  string                 `json:"taskId"`
	Score   float64                `json:"score"`
	Metrics map[string]MetricValue `json:"metrics,omitempty"`
	Detail  string                 `json:"detail,omitempty"`
}

type EvalContext struct {
	Environment string `json:"environment,omitempty"`
	Hardware    string `json:"hardware,omitempty"`
	StartedAt   string `json:"startedAt,omitempty"`
	CompletedAt string `json:"completedAt,omitempty"`
}

type EvalFilter struct {
	Category  EvalCategory
	Benchmark string
	Provider  string
}
