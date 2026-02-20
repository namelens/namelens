package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindBuiltInProfileOSS(t *testing.T) {
	profile, ok := FindBuiltInProfile("oss")
	require.True(t, ok)
	require.NotNil(t, profile)
	require.Equal(t, "oss", profile.Name)
	require.Empty(t, profile.TLDs)
	require.Equal(t, []string{"npm", "pypi", "cargo"}, profile.Registries)
	require.Equal(t, []string{"github"}, profile.Handles)
}

func TestFindBuiltInProfileReturnsCopy(t *testing.T) {
	profile, ok := FindBuiltInProfile("oss")
	require.True(t, ok)
	require.NotNil(t, profile)

	profile.Registries[0] = "changed"

	again, ok := FindBuiltInProfile("oss")
	require.True(t, ok)
	require.NotNil(t, again)
	require.Equal(t, "npm", again.Registries[0])
}
