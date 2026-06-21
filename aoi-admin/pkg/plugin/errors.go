package plugin

import "errors"

var (
	ErrDisabled             = errors.New("plugin host disabled")
	ErrPluginNotFound       = errors.New("plugin not found")
	ErrPluginOffline        = errors.New("plugin offline")
	ErrCapabilityNotFound   = errors.New("plugin capability not found")
	ErrProviderUnavailable  = errors.New("plugin provider unavailable")
	ErrTransportUnavailable = errors.New("plugin transport unavailable")
	ErrInvalidPlugin        = errors.New("invalid plugin metadata")
	ErrInvalidCapability    = errors.New("invalid plugin capability")
	ErrUnsupportedProtocol  = errors.New("unsupported plugin protocol")
	ErrUnauthorized         = errors.New("plugin request unauthorized")
	ErrUnsupportedAuth      = errors.New("unsupported plugin authentication")
)
