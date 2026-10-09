package cache

import (
	"github.com/omamx/profile-service/internal/domain"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestL1ExpiresAndDoesNotShareMutablePII(t *testing.T) {
	c, err := NewDualLayerCache("test", 10, nil)
	require.NoError(t, err)
	defer c.Close()
	original := &domain.CustomerProfile{FullName: "Original", Preferences: map[string]any{"dietary_preference": "NORMAL"}}
	c.l1Cache.Add("expired", profileEntry{original, time.Now().Add(-time.Second)})
	_, found := c.GetL1Direct("expired")
	require.False(t, found)
	c.l1Cache.Add("active", profileEntry{original, time.Now().Add(time.Second)})
	copy, found := c.GetL1Direct("active")
	require.True(t, found)
	copy.FullName = "Changed"
	copy.Preferences["dietary_preference"] = "VEGAN"
	again, found := c.GetL1Direct("active")
	require.True(t, found)
	require.Equal(t, "Original", again.FullName)
	require.Equal(t, "NORMAL", again.Preferences["dietary_preference"])
}
