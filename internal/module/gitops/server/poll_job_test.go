package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobToGitaly(t *testing.T) {
	tests := []struct {
		name              string
		glob              string
		expectedRepoPath  []byte
		expectedRecursive bool
	}{
		{
			name:              "full file name",
			glob:              "simple-path/manifest.yaml",
			expectedRepoPath:  []byte("simple-path"),
			expectedRecursive: false,
		},
		{
			name:              "empty",
			glob:              "",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: false,
		},
		{
			name:              "simple file1",
			glob:              "*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: false,
		},
		{
			name:              "files in directory1",
			glob:              "bla/*.yaml",
			expectedRepoPath:  []byte("bla"),
			expectedRecursive: false,
		},
		{
			name:              "recursive files in directory1",
			glob:              "bla/**/*.yaml",
			expectedRepoPath:  []byte("bla"),
			expectedRecursive: true,
		},
		{
			name:              "all files1",
			glob:              "**/*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: true,
		},
		{
			name:              "group1",
			glob:              "[a-z]*/*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: true,
		},
		{
			name:              "group2",
			glob:              "?bla/*.yaml",
			expectedRepoPath:  []byte{'.'},
			expectedRecursive: true,
		},
		{
			name:              "group3",
			glob:              "bla/?aaa/*.yaml",
			expectedRepoPath:  []byte("bla"),
			expectedRecursive: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { // nolint: scopelint
			gotRepoPath, gotRecursive := globToGitaly(tc.glob)  // nolint: scopelint
			assert.Equal(t, tc.expectedRepoPath, gotRepoPath)   // nolint: scopelint
			assert.Equal(t, tc.expectedRecursive, gotRecursive) // nolint: scopelint
		})
	}
}
