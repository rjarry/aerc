package models

// Commit represents a commit object in a revision control system.
type Commit struct {
	// ID is the commit hash.
	ID string
	// Subject is the subject line of the commit.
	Subject string
	// Author is the author's name.
	Author string
	// Date associated with the given commit.
	Date string
	// MessageId is the message id for the message that contains the commit
	// diff. This field is only set when commits were applied via patch
	// apply system.
	MessageId string
	// Tag is a user label that is assigned to one or multiple commits. It
	// creates a logical connection between a group of commits to represent
	// a patch set.
	Tag string
}

// Project contains the data to access a revision control system and to store
// the internal patch tracking data.
type Project struct {
	// Name is the project name and works as the project ID. Do not change
	// it.
	Name string
	// Root represents the root directory of the revision control system.
	Root string
	// RevctrlID stores the ID for the revision control system.
	RevctrlID string
	// Base represents the reference (base) commit.
	Base Commit
	// Commits contains the commits that are being tracked. The slice can
	// contain any commit between the Base commit and HEAD. These commits
	// will be updated by an applying, removing or rebase operation.
	Commits []Commit
}

// RevisionController is an interface to a revision control system.
type RevisionController interface {
	// Returns the commit hash of the HEAD commit.
	Head() (string, error)
	// History accepts a commit hash and returns a list of commit hashes
	// between the provided hash and HEAD. The order of the returned slice
	// is important. The commit hashes should be ordered from "earlier" to
	// "later" where the last element must be HEAD.
	History(string) ([]string, error)
	// Clean returns true if there are no unstaged changes. If there are
	// unstaged changes, applying and removing patches will not work.
	Clean() bool
	// Exists returns true if the commit hash exists in the commit history.
	Exists(string) bool
	// Subject returns the subject line for the provided commit hash.
	Subject(string) string
	// Author returns the author for the provided commit hash.
	Author(string) string
	// Date returns the date for the provided commit hash.
	Date(string) string
	// Remove removes the commit with the provided commit hash from the
	// repository.
	Remove(string) error
	// ApplyCmd returns a string with an executable command that is used to
	// apply patches with the :pipe command.
	ApplyCmd() string
}

// PersistentStorer is an interface to a persistent storage for Project structs.
type PersistentStorer interface {
	// StoreProject saves the project data persistently. If overwrite is
	// true, it will write over existing data.
	StoreProject(Project, bool) error
	// DeleteProject removes the project data from the store.
	DeleteProject(string) error
	// CurrentName returns the Project.Name for the active project.
	CurrentName() (string, error)
	// SetCurrent stores a Project.Name and make that project active.
	SetCurrent(string) error
	// Current returns the project data for the active project.
	Current() (Project, error)
	// Names returns a slice of Project.Name for all stored projects.
	Names() ([]string, error)
	// Projects returns all stored projects.
	Projects() ([]Project, error)
}
