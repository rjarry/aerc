package app

import (
	"strings"
	"testing"

	"git.sr.ht/~rjarry/aerc/config"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
)

// mockBackend implements types.Backend for testing
type mockBackend struct{}

func (m *mockBackend) Run()                               {}
func (m *mockBackend) Capabilities() *models.Capabilities { return nil }
func (m *mockBackend) PathSeparator() string              { return "/" }

// newTestTree creates a DirectoryTree with a mock backend for testing
func newTestTree() *DirectoryTree {
	worker := &types.Worker{Backend: &mockBackend{}}
	dirlist := &DirectoryList{
		worker:   worker,
		acctConf: &config.AccountConfig{Name: "test"},
	}
	return &DirectoryTree{
		DirectoryList: dirlist,
		listIdx:       -1,
	}
}

// buildTree constructs a tree from a list of folder paths.
// Parent-child relationships are inferred from path structure.
// Example: []string{"INBOX", "INBOX/Work", "INBOX/Personal", "Sent"}
// Creates:
//
//	├── INBOX (has children)
//	│   ├── Work (leaf)
//	│   └── Personal (leaf)
//	└── Sent (leaf)
func buildTree(folders []string) []*types.Thread {
	nodes := make(map[string]*types.Thread)
	var list []*types.Thread

	// Create all nodes first
	for _, path := range folders {
		nodes[path] = &types.Thread{Uid: models.UID(path)}
	}

	// Set up parent-child relationships
	for _, path := range folders {
		node := nodes[path]

		// Find parent by looking for longest matching prefix
		lastSep := strings.LastIndex(path, "/")
		if lastSep > 0 {
			parentPath := path[:lastSep]
			if parent, ok := nodes[parentPath]; ok {
				node.Parent = parent
				// Add as child (append to end of sibling chain)
				if parent.FirstChild == nil {
					parent.FirstChild = node
				} else {
					// Find last sibling
					sibling := parent.FirstChild
					for sibling.NextSibling != nil {
						sibling = sibling.NextSibling
					}
					sibling.NextSibling = node
					node.PrevSibling = sibling
				}
			}
		}

		list = append(list, node)
	}

	return list
}

func findNode(list []*types.Thread, uid string) *types.Thread {
	for _, node := range list {
		if string(node.Uid) == uid {
			return node
		}
	}
	return nil
}

func TestCollapseFolder(t *testing.T) {
	tests := []struct {
		name         string
		folders      []string
		folder       string
		setupHidden  map[string]int
		expectHidden map[string]int
	}{
		// Top-level folders
		{
			name:        "collapse top-level folder with children",
			folders:     []string{"INBOX", "INBOX/Work", "INBOX/Personal"},
			folder:      "INBOX",
			setupHidden: map[string]int{},
			expectHidden: map[string]int{
				"INBOX": 1,
			},
		},
		{
			name:         "collapse top-level leaf (no parent, no children)",
			folders:      []string{"INBOX", "Sent", "Trash"},
			folder:       "Sent",
			setupHidden:  map[string]int{},
			expectHidden: map[string]int{"Sent": 1},
		},

		// Nested leaf nodes
		{
			name:    "collapse leaf node collapses parent",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Personal"},
			folder:  "INBOX/Work",
			setupHidden: map[string]int{
				"INBOX": 0,
			},
			expectHidden: map[string]int{
				"INBOX": 1,
			},
		},
		{
			name:    "collapse deeply nested leaf collapses immediate parent",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Work/Projects", "INBOX/Work/Projects/Alpha"},
			folder:  "INBOX/Work/Projects/Alpha",
			setupHidden: map[string]int{
				"INBOX/Work/Projects": 0,
			},
			expectHidden: map[string]int{
				"INBOX/Work/Projects": 1,
			},
		},

		// Nested folders with children
		{
			name:    "collapse nested folder with children",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Work/Projects", "INBOX/Work/Archive"},
			folder:  "INBOX/Work",
			setupHidden: map[string]int{
				"INBOX/Work": 0,
			},
			expectHidden: map[string]int{
				"INBOX/Work": 1,
			},
		},

		// Already collapsed
		{
			name:    "collapse already collapsed node goes to parent",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Personal"},
			folder:  "INBOX",
			setupHidden: map[string]int{
				"INBOX": 1,
			},
			expectHidden: map[string]int{
				"INBOX": 1, // stays collapsed (no parent to collapse)
			},
		},

		// Non-existent folder
		{
			name:         "collapse non-existent folder does nothing",
			folders:      []string{"INBOX", "Sent"},
			folder:       "NonExistent",
			setupHidden:  map[string]int{},
			expectHidden: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := newTestTree()
			dt.list = buildTree(tt.folders)

			for uid, hidden := range tt.setupHidden {
				if node := findNode(dt.list, uid); node != nil {
					node.Hidden = hidden
				}
			}

			dt.CollapseFolder(tt.folder)

			for uid, expectedHidden := range tt.expectHidden {
				node := findNode(dt.list, uid)
				if node == nil {
					t.Errorf("node %s not found", uid)
					continue
				}
				if node.Hidden != expectedHidden {
					t.Errorf("node %s: got Hidden=%d, want %d",
						uid, node.Hidden, expectedHidden)
				}
			}
		})
	}
}

func TestExpandFolder(t *testing.T) {
	tests := []struct {
		name         string
		folders      []string
		folder       string
		setupHidden  map[string]int
		expectHidden map[string]int
	}{
		// Basic expand
		{
			name:    "expand collapsed top-level folder",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Personal"},
			folder:  "INBOX",
			setupHidden: map[string]int{
				"INBOX": 1,
			},
			expectHidden: map[string]int{
				"INBOX": 0,
			},
		},
		{
			name:    "expand already expanded folder is no-op",
			folders: []string{"INBOX", "INBOX/Work"},
			folder:  "INBOX",
			setupHidden: map[string]int{
				"INBOX": 0,
			},
			expectHidden: map[string]int{
				"INBOX": 0,
			},
		},

		// Nested folders
		{
			name:    "expand collapsed nested folder",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Work/Projects"},
			folder:  "INBOX/Work",
			setupHidden: map[string]int{
				"INBOX/Work": 1,
			},
			expectHidden: map[string]int{
				"INBOX/Work": 0,
			},
		},
		{
			name:    "expand deeply nested folder",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Work/Projects", "INBOX/Work/Projects/Alpha"},
			folder:  "INBOX/Work/Projects",
			setupHidden: map[string]int{
				"INBOX/Work/Projects": 1,
			},
			expectHidden: map[string]int{
				"INBOX/Work/Projects": 0,
			},
		},

		// Expand doesn't affect ancestors
		{
			name:    "expand does not affect collapsed ancestors",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Work/Projects"},
			folder:  "INBOX/Work/Projects",
			setupHidden: map[string]int{
				"INBOX":               1,
				"INBOX/Work":          1,
				"INBOX/Work/Projects": 1,
			},
			expectHidden: map[string]int{
				"INBOX":               1, // unchanged
				"INBOX/Work":          1, // unchanged
				"INBOX/Work/Projects": 0, // expanded
			},
		},

		// Non-existent
		{
			name:         "expand non-existent folder does nothing",
			folders:      []string{"INBOX"},
			folder:       "NonExistent",
			setupHidden:  map[string]int{},
			expectHidden: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := newTestTree()
			dt.list = buildTree(tt.folders)

			for uid, hidden := range tt.setupHidden {
				if node := findNode(dt.list, uid); node != nil {
					node.Hidden = hidden
				}
			}

			dt.ExpandFolder(tt.folder)

			for uid, expectedHidden := range tt.expectHidden {
				node := findNode(dt.list, uid)
				if node == nil {
					t.Errorf("node %s not found", uid)
					continue
				}
				if node.Hidden != expectedHidden {
					t.Errorf("node %s: got Hidden=%d, want %d",
						uid, node.Hidden, expectedHidden)
				}
			}
		})
	}
}

func TestToggleFolder(t *testing.T) {
	tests := []struct {
		name         string
		folders      []string
		folder       string
		setupHidden  map[string]int
		expectHidden map[string]int
	}{
		// Top-level with children
		{
			name:    "toggle expanded top-level with children collapses",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Personal"},
			folder:  "INBOX",
			setupHidden: map[string]int{
				"INBOX": 0,
			},
			expectHidden: map[string]int{
				"INBOX": 1,
			},
		},
		{
			name:    "toggle collapsed top-level with children expands",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Personal"},
			folder:  "INBOX",
			setupHidden: map[string]int{
				"INBOX": 1,
			},
			expectHidden: map[string]int{
				"INBOX": 0,
			},
		},

		// Top-level leaf (no parent, no children)
		{
			name:         "toggle top-level leaf is no-op",
			folders:      []string{"INBOX", "Sent", "Trash"},
			folder:       "Sent",
			setupHidden:  map[string]int{},
			expectHidden: map[string]int{"Sent": 0},
		},

		// Nested leaf nodes
		{
			name:    "toggle leaf with expanded parent collapses parent",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Personal"},
			folder:  "INBOX/Work",
			setupHidden: map[string]int{
				"INBOX": 0,
			},
			expectHidden: map[string]int{
				"INBOX": 1,
			},
		},
		{
			name:    "toggle leaf with collapsed parent expands parent",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Personal"},
			folder:  "INBOX/Work",
			setupHidden: map[string]int{
				"INBOX": 1,
			},
			expectHidden: map[string]int{
				"INBOX": 0,
			},
		},

		// Deeply nested leaf
		{
			name:    "toggle deeply nested leaf with expanded parent",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Work/Projects", "INBOX/Work/Projects/Alpha"},
			folder:  "INBOX/Work/Projects/Alpha",
			setupHidden: map[string]int{
				"INBOX/Work/Projects": 0,
			},
			expectHidden: map[string]int{
				"INBOX/Work/Projects": 1,
			},
		},
		{
			name:    "toggle deeply nested leaf with collapsed parent",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Work/Projects", "INBOX/Work/Projects/Alpha"},
			folder:  "INBOX/Work/Projects/Alpha",
			setupHidden: map[string]int{
				"INBOX/Work/Projects": 1,
			},
			expectHidden: map[string]int{
				"INBOX/Work/Projects": 0,
			},
		},

		// Nested folder with children
		{
			name:    "toggle expanded nested folder with children",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Work/Projects", "INBOX/Work/Archive"},
			folder:  "INBOX/Work",
			setupHidden: map[string]int{
				"INBOX/Work": 0,
			},
			expectHidden: map[string]int{
				"INBOX/Work": 1,
			},
		},
		{
			name:    "toggle collapsed nested folder with children",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Work/Projects", "INBOX/Work/Archive"},
			folder:  "INBOX/Work",
			setupHidden: map[string]int{
				"INBOX/Work": 1,
			},
			expectHidden: map[string]int{
				"INBOX/Work": 0,
			},
		},

		// Toggle doesn't affect siblings or ancestors
		{
			name:    "toggle only affects target node not siblings",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Personal", "Drafts", "Drafts/Important"},
			folder:  "INBOX",
			setupHidden: map[string]int{
				"INBOX":  0,
				"Drafts": 1,
			},
			expectHidden: map[string]int{
				"INBOX":  1, // toggled
				"Drafts": 1, // unchanged
			},
		},

		// Non-existent
		{
			name:         "toggle non-existent folder does nothing",
			folders:      []string{"INBOX"},
			folder:       "NonExistent",
			setupHidden:  map[string]int{},
			expectHidden: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := newTestTree()
			dt.list = buildTree(tt.folders)

			for uid, hidden := range tt.setupHidden {
				if node := findNode(dt.list, uid); node != nil {
					node.Hidden = hidden
				}
			}

			dt.ToggleFolder(tt.folder)

			for uid, expectedHidden := range tt.expectHidden {
				node := findNode(dt.list, uid)
				if node == nil {
					t.Errorf("node %s not found", uid)
					continue
				}
				if node.Hidden != expectedHidden {
					t.Errorf("node %s: got Hidden=%d, want %d",
						uid, node.Hidden, expectedHidden)
				}
			}
		})
	}
}

// TestToggleFolderIdempotency verifies that toggle-toggle returns to original state
func TestToggleFolderIdempotency(t *testing.T) {
	tests := []struct {
		name        string
		folders     []string
		folder      string
		setupHidden map[string]int
	}{
		{
			name:    "double toggle on folder with children",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Personal"},
			folder:  "INBOX",
			setupHidden: map[string]int{
				"INBOX": 0,
			},
		},
		{
			name:    "double toggle on collapsed folder",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Work/Projects"},
			folder:  "INBOX/Work",
			setupHidden: map[string]int{
				"INBOX/Work": 1,
			},
		},
		{
			name:    "double toggle on deeply nested folder",
			folders: []string{"INBOX", "INBOX/Work", "INBOX/Work/Projects", "INBOX/Work/Projects/Alpha"},
			folder:  "INBOX/Work/Projects",
			setupHidden: map[string]int{
				"INBOX/Work/Projects": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := newTestTree()
			dt.list = buildTree(tt.folders)

			// Setup and record initial state
			initialState := make(map[string]int)
			for uid, hidden := range tt.setupHidden {
				if node := findNode(dt.list, uid); node != nil {
					node.Hidden = hidden
					initialState[uid] = hidden
				}
			}

			// Toggle twice
			dt.ToggleFolder(tt.folder)
			dt.ToggleFolder(tt.folder)

			// Verify we're back to initial state
			for uid, expectedHidden := range initialState {
				node := findNode(dt.list, uid)
				if node == nil {
					t.Errorf("node %s not found", uid)
					continue
				}
				if node.Hidden != expectedHidden {
					t.Errorf("after double toggle, node %s: got Hidden=%d, want %d",
						uid, node.Hidden, expectedHidden)
				}
			}
		})
	}
}
