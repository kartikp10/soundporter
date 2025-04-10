package adapters

import (
	"fmt"
)

// BaseAdapter provides common functionality for platform adapters
type BaseAdapter struct {
	authenticated bool
	platformName  string
}

// NewBaseAdapter creates a new BaseAdapter
func NewBaseAdapter(platformName string) BaseAdapter {
	return BaseAdapter{
		authenticated: false,
		platformName:  platformName,
	}
}

// SetAuthenticated updates the authentication status
func (b *BaseAdapter) SetAuthenticated(status bool) {
	b.authenticated = status
}

// IsAuthenticated checks if the adapter is authenticated
func (b *BaseAdapter) IsAuthenticated() bool {
	return b.authenticated
}

// CheckAuth ensures the adapter is authenticated before making API calls
func (b *BaseAdapter) CheckAuth() error {
	if !b.IsAuthenticated() {
		return fmt.Errorf("not authenticated, call Authenticate() first for %s", b.platformName)
	}
	return nil
}

// PlatformName returns the name of the platform
func (b *BaseAdapter) PlatformName() string {
	return b.platformName
}
