package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
)

type Searchable interface {
	PerformSearch()
	FuzzyMatch(text, query string) bool
	GetSearchQuery() string
	SetSearchQuery(input string)
}

type Scrollable interface {
	AdjustScrolling(itemCount int)
	GetCurrentIndex() int
	SetCurrentIndex(int)
	GetScrollOffset() int
}

type ItemProvider interface {
	GetItems() []string
	GetItemCount() int
}

type ListComponent struct {
	currentIndex int
	scrollOffset int
	visibleLines int
	width        int
	height       int

	searchInput     textinput.Model
	searchQuery     string
	filteredIndices []int
	searchSelected  int

	mode Mode
}

func (lc ListComponent) GetSerachQuery() string {
	return lc.searchQuery
}

func (lc ListComponent) SetSearchQuery(query string) {
	lc.searchQuery = query
}

func (lc ListComponent) FuzzyMatch(text, query string) bool {
	if query == "" {
		return true
	}

	textIdx := 0
	for _, queryChar := range query {
		found := false
		for textIdx < len(text) {
			if rune(text[textIdx]) == queryChar {
				found = true
				textIdx++
				break
			}
			textIdx++
		}
		if !found {
			return false
		}
	}
	return true
}

func (lc ListComponent) PerformSearch(ItemProvider ItemProvider) {
	if lc.searchQuery == "" {
		lc.filteredIndices = nil
		lc.searchSelected = 0
		return
	}

	query := strings.ToLower(lc.searchQuery)
	lc.filteredIndices = []int{}

	items := ItemProvider.GetItems()

	for i, item := range items {
		if lc.FuzzyMatch(strings.ToLower(item), query) {
			lc.filteredIndices = append(lc.filteredIndices, i)
		}
	}

	lc.searchSelected = 0
}
