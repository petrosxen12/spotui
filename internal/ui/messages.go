package ui

import "github.com/petrosxen/spotui/internal/app"

type connectionMsg struct {
	user app.User
	err  error
}

type searchMsg struct {
	query   string
	results app.Results
	err     error
}

type playbackMsg struct {
	state app.PlaybackState
	err   error
}

type actionMsg struct {
	text string
	err  error
}

type devicesMsg struct {
	devices     []app.Device
	err         error
	pushHistory bool
}

type deviceCacheMsg struct {
	devices []app.Device
	err     error
}

type deviceSelectedMsg struct {
	device app.Device
	err    error
}

type helpMsg struct{}

type localPlayerStatusMsg struct {
	status localPlayerStatus
	err    error
}

type localPlayerActionMsg struct {
	text   string
	status localPlayerStatus
	err    error
}

type accentColorMsg struct {
	albumArtURL string
	color       string
	err         error
}

type pollTickMsg struct{}

type bootAnimationMsg struct{}
