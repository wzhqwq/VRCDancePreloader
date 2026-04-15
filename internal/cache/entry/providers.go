package entry

import (
	"context"
)

type UrlResourceProvider func(id string, ctx context.Context) (*RemoteVideoInfo, error)

var resolverProvider func(id string) UrlResolver

func SetResolverProvider(provider func(id string) UrlResolver) {
	resolverProvider = provider
}

func NewEntry(id string) Entry {
	resolver := resolverProvider(id)
	if resolver == nil {
		return nil
	}

	return NewUrlBasedEntry(id, resolver)
}
