package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTagFinderMatch(t *testing.T) {
	// Given
	finder, err := createTagFinder()
	assert.NoError(t, err)

	// When
	tags := findTags(finder, "foo #bar baz")

	// Then
	assert.Len(t, tags, 1)
	assert.Contains(t, tags, "bar")
}

func TestTagFinderMultiMatch(t *testing.T) {
	// Given
	finder, err := createTagFinder()
	assert.NoError(t, err)

	// When
	tags := findTags(finder, "foo #bar baz #qux")

	// Then
	assert.Len(t, tags, 2)
	assert.Contains(t, tags, "bar")
	assert.Contains(t, tags, "qux")
}

func TestTagFinderNoMatch(t *testing.T) {
	// Given
	finder, err := createTagFinder()
	assert.NoError(t, err)

	// When
	tags := findTags(finder, "No tags here, mate!")

	// Then
	assert.Len(t, tags, 0)
}
