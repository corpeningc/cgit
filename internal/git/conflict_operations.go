package git

type ResolutionChoice int

const (
	ChooseOurs ResolutionChoice = iota
	ChooseTheirs
	ChooseBoth
	ManualEdit
)

type ConflictFile struct {
	Path      string
	Conflicts []ConflictSection
}

type ConflictSection struct {
	StartLine    int
	EndLine      int
	TheirChanges string
	OurChanges   string
	BaseContent  string
}

// GetConflictedFiles() - Returns list of files with conflicts
// ParseConflictMarkers() - Parses <<<<<<, =======, >>>>>> markers
// ResolveConflict() - Writes resolved content back to file
// AcceptOurs() - Keep current branch changes
// AcceptTheirs() - Keep incoming branch changes
// AcceptBoth() - Merge both changes
