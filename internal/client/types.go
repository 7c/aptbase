package client

// Version is the response from GET /api/version.
type Version struct {
	Version string `json:"Version"`
}

// Repo is a local repository (GET /api/repos).
type Repo struct {
	Name                string `json:"Name"`
	Comment             string `json:"Comment"`
	DefaultDistribution string `json:"DefaultDistribution"`
	DefaultComponent    string `json:"DefaultComponent"`
}

// CreateRepoRequest is the body for POST /api/repos.
type CreateRepoRequest struct {
	Name                string `json:"Name"`
	Comment             string `json:"Comment,omitempty"`
	DefaultDistribution string `json:"DefaultDistribution,omitempty"`
	DefaultComponent    string `json:"DefaultComponent,omitempty"`
}

// EditRepoRequest is the body for PUT /api/repos/:name.
type EditRepoRequest struct {
	Comment             string `json:"Comment,omitempty"`
	DefaultDistribution string `json:"DefaultDistribution,omitempty"`
	DefaultComponent    string `json:"DefaultComponent,omitempty"`
}

// PublishedRepo is an entry from GET /api/publish.
type PublishedRepo struct {
	Storage       string             `json:"Storage"`
	Prefix        string             `json:"Prefix"`
	Distribution  string             `json:"Distribution"`
	SourceKind    string             `json:"SourceKind"`
	Sources       []PublishedSource  `json:"Sources"`
	Architectures []string           `json:"Architectures"`
	Label         string             `json:"Label"`
	Origin        string             `json:"Origin"`
}

// PublishedSource is a component/source pair within a publication.
type PublishedSource struct {
	Component string `json:"Component"`
	Name      string `json:"Name"`
}

// Signing controls GPG signing options on publish operations.
type Signing struct {
	Skip       bool   `json:"Skip,omitempty"`
	GpgKey     string `json:"GpgKey,omitempty"`
	Keyring    string `json:"Keyring,omitempty"`
	Passphrase string `json:"Passphrase,omitempty"`
	Batch      bool   `json:"Batch,omitempty"`
}

// UpdatePublishRequest is the body for PUT /api/publish/:prefix/:distribution.
type UpdatePublishRequest struct {
	Signing        Signing `json:"Signing"`
	ForceOverwrite bool    `json:"ForceOverwrite,omitempty"`
}

// Task is an asynchronous aptly task (returned when ?_async=1 is used).
type Task struct {
	ID    int    `json:"ID"`
	Name  string `json:"Name"`
	State int    `json:"State"`
}

// Task states reported by aptly.
const (
	TaskInit = iota
	TaskRunning
	TaskSucceeded
	TaskFailed
)

// Done reports whether the task has reached a terminal state.
func (t Task) Done() bool { return t.State == TaskSucceeded || t.State == TaskFailed }

// Failed reports whether the task ended in failure.
func (t Task) Failed() bool { return t.State == TaskFailed }

// StateString renders a task state as a human label.
func (t Task) StateString() string {
	switch t.State {
	case TaskInit:
		return "init"
	case TaskRunning:
		return "running"
	case TaskSucceeded:
		return "succeeded"
	case TaskFailed:
		return "failed"
	default:
		return "unknown"
	}
}
